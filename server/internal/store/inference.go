package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/router"
)

var (
	ErrInsufficientBalance = errors.New("insufficient confirmed session balance")
	ErrIneligibleRoute     = errors.New("route is no longer eligible")
)

type RoutingState struct {
	MachineID, OfferID, Model                      string
	Capabilities                                   []string
	PriceVersion, MaximumCost, LatencyMilliseconds uint64
	ConfirmedBond, Healthy                         bool
	Capacity                                       uint32
	SuccessBasisPoints                             uint16
	Reputation                                     uint64
}

type InferenceReservation struct {
	RequestID, SessionID, AccountID, MachineID, OfferID string
	PriceVersion, MaximumSpend                          uint64
}

type ReceiptProposal struct {
	RequestID, SessionID, MachineID, OfferID, Model string
	PriceVersion                                    uint64
	InputTokens, OutputTokens, ComputeMilliseconds  uint64
	InputHash, OutputHash                           [32]byte
	CompletedAt                                     time.Time
}

func (s *Store) UpsertRoutingState(ctx context.Context, state RoutingState) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO provider_routing_state (machine_id, offer_id, model, capabilities, price_version, confirmed_bond, healthy, capacity, maximum_cost, latency_milliseconds, success_basis_points, reputation)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (machine_id, offer_id) DO UPDATE SET model=EXCLUDED.model, capabilities=EXCLUDED.capabilities, price_version=EXCLUDED.price_version, confirmed_bond=EXCLUDED.confirmed_bond, healthy=EXCLUDED.healthy, capacity=EXCLUDED.capacity, maximum_cost=EXCLUDED.maximum_cost, latency_milliseconds=EXCLUDED.latency_milliseconds, success_basis_points=EXCLUDED.success_basis_points, reputation=EXCLUDED.reputation, updated_at=now()`, state.MachineID, state.OfferID, state.Model, state.Capabilities, state.PriceVersion, state.ConfirmedBond, state.Healthy, state.Capacity, decimal(state.MaximumCost), state.LatencyMilliseconds, state.SuccessBasisPoints, state.Reputation)
	return err
}

func (s *Store) RoutingCandidates(ctx context.Context, model string) ([]router.Candidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT machine_id, offer_id, model, array_to_json(capabilities), price_version, confirmed_bond, healthy, capacity, maximum_cost::text, latency_milliseconds, success_basis_points, reputation FROM provider_routing_state WHERE model=$1`, model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []router.Candidate
	for rows.Next() {
		var candidate router.Candidate
		var maximum, capabilities string
		if err := rows.Scan(&candidate.MachineID, &candidate.OfferID, &candidate.Model, &capabilities, &candidate.PriceVersion, &candidate.ConfirmedBond, &candidate.Healthy, &candidate.Capacity, &maximum, &candidate.LatencyMilliseconds, &candidate.SuccessBasisPoints, &candidate.Reputation); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(capabilities), &candidate.Capabilities); err != nil {
			return nil, err
		}
		candidate.MaximumCost, err = strconv.ParseUint(maximum, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("routing cost exceeds broker limit: %w", err)
		}
		result = append(result, candidate)
	}
	return result, rows.Err()
}

func (s *Store) UpdateProviderCapacity(ctx context.Context, machineID string, capacity v1.Capacity) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE provider_routing_state SET capacity=0, updated_at=now() WHERE machine_id=$1`, machineID); err != nil {
		return err
	}
	for _, offer := range capacity.Offers {
		available := uint32(1)
		if capacity.Available == 0 {
			available = 0
		}
		if _, err := tx.ExecContext(ctx, `UPDATE provider_routing_state SET capacity=GREATEST(0, $4 - (SELECT count(*) FROM inference_reservations r JOIN requests q ON q.id=r.request_id WHERE q.machine_id=$1 AND q.offer_id=$2 AND r.released_at IS NULL)), healthy=true, updated_at=now() WHERE machine_id=$1 AND offer_id=$2 AND price_version=$3`, machineID, offer.OfferID, offer.PriceVersion, available); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ReconcileProviderCapacity(ctx context.Context, machineID string, capacity v1.Capacity, chainID uint64, contractAddress string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE provider_routing_state SET capacity=0,updated_at=now() WHERE machine_id=$1`, machineID); err != nil {
		return err
	}
	var wallet string
	if err := tx.QueryRowContext(ctx, `SELECT a.wallet_address FROM machines m JOIN accounts a ON a.id=m.account_id WHERE m.id=$1 AND m.revoked_at IS NULL`, machineID).Scan(&wallet); err != nil {
		return err
	}
	for _, offered := range capacity.Offers {
		if offered.OfferHash == "" {
			continue
		}
		var input, output, compute, bond string
		var exit uint64
		err := tx.QueryRowContext(ctx, `SELECT o.input_per_million::text,o.output_per_million::text,o.compute_per_second::text,a.provider_bond::text,a.bond_exit_available_at FROM chain_offers o JOIN chain_accounts a ON a.chain_id=o.chain_id AND a.contract_address=o.contract_address AND lower(a.address)=lower(o.provider) WHERE o.chain_id=$1 AND lower(o.contract_address)=lower($2) AND lower(o.provider)=lower($3) AND o.offer_id=$4 AND o.version=$5 AND o.model_hash=$6 AND o.capability_hash=$7`, chainID, contractAddress, wallet, offered.OfferHash, offered.PriceVersion, offered.ModelHash, offered.CapabilityHash).Scan(&input, &output, &compute, &bond, &exit)
		if err != nil || bond == "0" || exit != 0 {
			return ErrIneligibleRoute
		}
		inputValue, e1 := strconv.ParseUint(input, 10, 64)
		outputValue, e2 := strconv.ParseUint(output, 10, 64)
		computeValue, e3 := strconv.ParseUint(compute, 10, 64)
		if e1 != nil || e2 != nil || e3 != nil || inputValue > ^uint64(0)-outputValue || inputValue+outputValue > ^uint64(0)-computeValue {
			return ErrIneligibleRoute
		}
		var backendID string
		if err := tx.QueryRowContext(ctx, `INSERT INTO backends(id,machine_id,kind,model,enabled) VALUES ($1,$2,$3,$4,true) ON CONFLICT (machine_id,kind,model) DO UPDATE SET enabled=true RETURNING id`, `backend:`+machineID+":"+offered.OfferID, machineID, offered.BackendKind, offered.Model).Scan(&backendID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO offers(id,backend_id,version,input_per_million,output_per_million,compute_per_second) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id,version) DO NOTHING`, offered.OfferID, backendID, offered.PriceVersion, input, output, compute); err != nil {
			return err
		}
		available := uint32(0)
		if capacity.Available > 0 {
			available = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO provider_routing_state(machine_id,offer_id,model,capabilities,price_version,confirmed_bond,healthy,capacity,maximum_cost) VALUES ($1,$2,$3,$4,$5,true,true,$6,$7) ON CONFLICT(machine_id,offer_id) DO UPDATE SET model=EXCLUDED.model,capabilities=EXCLUDED.capabilities,price_version=EXCLUDED.price_version,confirmed_bond=true,healthy=true,capacity=EXCLUDED.capacity,maximum_cost=EXCLUDED.maximum_cost,updated_at=now()`, machineID, offered.OfferID, offered.Model, []string{"text", "stream"}, offered.PriceVersion, available, decimal(inputValue+outputValue+computeValue)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) OpenSession(ctx context.Context, accountID string) (string, uint64, error) {
	var id, balance string
	err := s.db.QueryRowContext(ctx, `SELECT id, confirmed_balance_wei::text FROM sessions WHERE account_id=$1 AND state='open' ORDER BY created_at LIMIT 1`, accountID).Scan(&id, &balance)
	if err != nil {
		return "", 0, err
	}
	value, err := strconv.ParseUint(balance, 10, 64)
	return id, value, err
}

func (s *Store) ReserveInference(ctx context.Context, reservation InferenceReservation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, reservation.SessionID); err != nil {
		return err
	}
	var balance, reserved string
	var accountID, state string
	if err := tx.QueryRowContext(ctx, `SELECT account_id, state, confirmed_balance_wei::text FROM sessions WHERE id=$1 FOR UPDATE`, reservation.SessionID).Scan(&accountID, &state, &balance); err != nil {
		return err
	}
	if accountID != reservation.AccountID || state != "open" {
		return ErrInsufficientBalance
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(sum(amount),0)::text FROM inference_reservations WHERE session_id=$1 AND released_at IS NULL`, reservation.SessionID).Scan(&reserved); err != nil {
		return err
	}
	confirmed, err := strconv.ParseUint(balance, 10, 64)
	if err != nil {
		return err
	}
	locked, err := strconv.ParseUint(reserved, 10, 64)
	if err != nil || locked > confirmed || reservation.MaximumSpend > confirmed-locked {
		return ErrInsufficientBalance
	}
	result, err := tx.ExecContext(ctx, `UPDATE provider_routing_state SET capacity=capacity-1, updated_at=now() WHERE machine_id=$1 AND offer_id=$2 AND price_version=$3 AND confirmed_bond AND healthy AND capacity>0 AND maximum_cost <= $4`, reservation.MachineID, reservation.OfferID, reservation.PriceVersion, decimal(reservation.MaximumSpend))
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return ErrIneligibleRoute
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO requests (id, session_id, state, machine_id, offer_id, price_version, maximum_spend) VALUES ($1,$2,'created',$3,$4,$5,$6)`, reservation.RequestID, reservation.SessionID, reservation.MachineID, reservation.OfferID, reservation.PriceVersion, decimal(reservation.MaximumSpend)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO inference_reservations (request_id, session_id, amount) VALUES ($1,$2,$3)`, reservation.RequestID, reservation.SessionID, decimal(reservation.MaximumSpend)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE requests SET state='reserved', updated_at=now() WHERE id=$1`, reservation.RequestID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request',$1,'request.reserved',jsonb_build_object('from','created','to','reserved'))`, reservation.RequestID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE requests SET state='offered', updated_at=now() WHERE id=$1`, reservation.RequestID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request',$1,'request.offered',jsonb_build_object('from','reserved','to','offered'))`, reservation.RequestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CompleteInference(ctx context.Context, proposal ReceiptProposal) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE requests SET state='completed', updated_at=now() WHERE id=$1 AND session_id=$2 AND state='streaming'`, proposal.RequestID, proposal.SessionID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM receipt_proposals WHERE request_id=$1)`, proposal.RequestID).Scan(&exists); err == nil && exists {
			return nil
		}
		return ErrInvalidTransition
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO receipt_proposals (request_id,session_id,machine_id,offer_id,model,price_version,input_tokens,output_tokens,compute_milliseconds,input_hash,output_hash,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, proposal.RequestID, proposal.SessionID, proposal.MachineID, proposal.OfferID, proposal.Model, proposal.PriceVersion, proposal.InputTokens, proposal.OutputTokens, proposal.ComputeMilliseconds, proposal.InputHash[:], proposal.OutputHash[:], proposal.CompletedAt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE inference_reservations SET released_at=now() WHERE request_id=$1 AND released_at IS NULL`, proposal.RequestID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE provider_routing_state SET capacity=capacity+1, updated_at=now() WHERE machine_id=$1 AND offer_id=$2`, proposal.MachineID, proposal.OfferID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request',$1,'request.completed',jsonb_build_object('from','streaming','to','completed'))`, proposal.RequestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AbortInference(ctx context.Context, requestID, terminalState string) error {
	if terminalState != "failed" && terminalState != "cancelled" {
		return ErrInvalidTransition
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var machineID, offerID, state string
	err = tx.QueryRowContext(ctx, `SELECT machine_id, offer_id, state FROM requests WHERE id=$1 FOR UPDATE`, requestID).Scan(&machineID, &offerID, &state)
	if err != nil {
		return err
	}
	if state == terminalState {
		return nil
	}
	if !transitions[state][terminalState] {
		return ErrInvalidTransition
	}
	if _, err := tx.ExecContext(ctx, `UPDATE requests SET state=$2, updated_at=now() WHERE id=$1`, requestID, terminalState); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE inference_reservations SET released_at=now() WHERE request_id=$1 AND released_at IS NULL`, requestID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 1 {
		if _, err := tx.ExecContext(ctx, `UPDATE provider_routing_state SET capacity=capacity+1, updated_at=now() WHERE machine_id=$1 AND offer_id=$2`, machineID, offerID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request',$1,$2,jsonb_build_object('from',$3,'to',$4))`, requestID, "request."+terminalState, state, terminalState); err != nil {
		return err
	}
	return tx.Commit()
}

func decimal(value uint64) string { return strconv.FormatUint(value, 10) }
