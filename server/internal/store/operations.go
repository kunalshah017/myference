package store

import (
	"context"
	"database/sql"
	"strings"
)

type AccountOperations struct {
	ChainID             uint64             `json:"chain_id"`
	ContractAddress     string             `json:"contract_address"`
	ExplorerURL         string             `json:"explorer_url"`
	Confirmations       uint64             `json:"confirmations"`
	WalletAddress       string             `json:"wallet_address"`
	CustomerBalanceWei  string             `json:"customer_balance_wei"`
	ProviderBondWei     string             `json:"provider_bond_wei"`
	ClaimableWei        string             `json:"claimable_wei"`
	ProviderEarningsWei string             `json:"provider_earnings_wei"`
	BondExitAvailableAt uint64             `json:"bond_exit_available_at"`
	Sessions            []OperationSession `json:"sessions"`
	Machines            []OperationMachine `json:"machines"`
	Offers              []OperationOffer   `json:"offers"`
}

type OperationSession struct {
	SessionID        string `json:"session_id"`
	AllowanceWei     string `json:"allowance_wei"`
	SpentWei         string `json:"spent_wei"`
	ExpiresAt        uint64 `json:"expires_at"`
	CloseAvailableAt uint64 `json:"close_available_at"`
	Finalized        bool   `json:"finalized"`
}

type OperationMachine struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Revoked  bool               `json:"revoked"`
	Backends []OperationBackend `json:"backends"`
}

type OperationBackend struct {
	ID          string   `json:"id"`
	OfferHashes []string `json:"offer_hashes"`
	Kind        string   `json:"kind"`
	Model       string   `json:"model"`
	Enabled     bool     `json:"enabled"`
	Healthy     bool     `json:"healthy"`
	Capacity    uint32   `json:"capacity"`
}

type OperationOffer struct {
	OfferID             string `json:"offer_id"`
	ModelHash           string `json:"model_hash"`
	CapabilityHash      string `json:"capability_hash"`
	Version             uint64 `json:"version"`
	InputPerMillionWei  string `json:"input_per_million_wei"`
	OutputPerMillionWei string `json:"output_per_million_wei"`
	ComputePerSecondWei string `json:"compute_per_second_wei"`
}

func (s *Store) AccountOperations(ctx context.Context, accountID string, chainID uint64, contractAddress, explorerURL string, confirmations uint64) (AccountOperations, error) {
	view := AccountOperations{ChainID: chainID, ContractAddress: contractAddress, ExplorerURL: explorerURL, Confirmations: confirmations, CustomerBalanceWei: "0", ProviderBondWei: "0", ClaimableWei: "0", ProviderEarningsWei: "0", Sessions: []OperationSession{}, Machines: []OperationMachine{}, Offers: []OperationOffer{}}
	if err := s.db.QueryRowContext(ctx, `SELECT wallet_address FROM accounts WHERE id=$1`, accountID).Scan(&view.WalletAddress); err != nil {
		return AccountOperations{}, err
	}
	err := s.db.QueryRowContext(ctx, `SELECT customer_balance::text,provider_bond::text,claimable::text,bond_exit_available_at FROM chain_accounts WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(address)=lower($3)`, chainID, contractAddress, view.WalletAddress).Scan(&view.CustomerBalanceWei, &view.ProviderBondWei, &view.ClaimableWei, &view.BondExitAvailableAt)
	if err != nil && err != sql.ErrNoRows {
		return AccountOperations{}, err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(sum(provider_amount),0)::text FROM chain_settlements WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3)`, chainID, contractAddress, view.WalletAddress).Scan(&view.ProviderEarningsWei)

	sessionRows, err := s.db.QueryContext(ctx, `SELECT session_id,allowance::text,spent::text,expires_at,close_available_at,finalized FROM chain_sessions WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(customer)=lower($3) ORDER BY expires_at DESC`, chainID, contractAddress, view.WalletAddress)
	if err != nil {
		return AccountOperations{}, err
	}
	for sessionRows.Next() {
		var item OperationSession
		if err := sessionRows.Scan(&item.SessionID, &item.AllowanceWei, &item.SpentWei, &item.ExpiresAt, &item.CloseAvailableAt, &item.Finalized); err != nil {
			sessionRows.Close()
			return AccountOperations{}, err
		}
		view.Sessions = append(view.Sessions, item)
	}
	if err := sessionRows.Close(); err != nil {
		return AccountOperations{}, err
	}

	machineRows, err := s.db.QueryContext(ctx, `SELECT m.id,m.name,m.revoked_at IS NOT NULL,b.id,b.kind,b.model,b.enabled,COALESCE(prs.healthy,false),COALESCE(prs.capacity,0),COALESCE(prs.offer_hashes,'') FROM machines m LEFT JOIN backends b ON b.machine_id=m.id LEFT JOIN LATERAL (SELECT bool_or(healthy) AS healthy,COALESCE(sum(capacity),0) AS capacity,string_agg(DISTINCT offer_hash,',' ORDER BY offer_hash) FILTER (WHERE confirmed_bond AND healthy AND capacity>0) AS offer_hashes FROM provider_routing_state WHERE machine_id=m.id AND model=b.model AND backend_kind=b.kind) prs ON true WHERE m.account_id=$1 ORDER BY m.created_at,b.id`, accountID)
	if err != nil {
		return AccountOperations{}, err
	}
	indices := map[string]int{}
	for machineRows.Next() {
		var machineID, name string
		var revoked bool
		var backendID, kind, model sql.NullString
		var enabled, healthy sql.NullBool
		var capacity sql.NullInt64
		var offerHashes string
		if err := machineRows.Scan(&machineID, &name, &revoked, &backendID, &kind, &model, &enabled, &healthy, &capacity, &offerHashes); err != nil {
			machineRows.Close()
			return AccountOperations{}, err
		}
		index, ok := indices[machineID]
		if !ok {
			index = len(view.Machines)
			indices[machineID] = index
			view.Machines = append(view.Machines, OperationMachine{ID: machineID, Name: name, Revoked: revoked, Backends: []OperationBackend{}})
		}
		if backendID.Valid {
			activeOffers := []string{}
			if offerHashes != "" {
				activeOffers = strings.Split(offerHashes, ",")
			}
			view.Machines[index].Backends = append(view.Machines[index].Backends, OperationBackend{ID: backendID.String, OfferHashes: activeOffers, Kind: kind.String, Model: model.String, Enabled: enabled.Bool, Healthy: healthy.Bool, Capacity: uint32(capacity.Int64)})
		}
	}
	if err := machineRows.Close(); err != nil {
		return AccountOperations{}, err
	}

	offerRows, err := s.db.QueryContext(ctx, `SELECT offer_id,version,model_hash,capability_hash,input_per_million::text,output_per_million::text,compute_per_second::text FROM chain_offers WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3) ORDER BY offer_id,version DESC`, chainID, contractAddress, view.WalletAddress)
	if err != nil {
		return AccountOperations{}, err
	}
	for offerRows.Next() {
		var item OperationOffer
		if err := offerRows.Scan(&item.OfferID, &item.Version, &item.ModelHash, &item.CapabilityHash, &item.InputPerMillionWei, &item.OutputPerMillionWei, &item.ComputePerSecondWei); err != nil {
			offerRows.Close()
			return AccountOperations{}, err
		}
		view.Offers = append(view.Offers, item)
	}
	if err := offerRows.Close(); err != nil {
		return AccountOperations{}, err
	}
	view.ContractAddress = strings.ToLower(view.ContractAddress)
	return view, nil
}
