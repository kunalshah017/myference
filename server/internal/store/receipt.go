package store

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

type ReceiptDomain struct {
	ChainID          uint64
	ContractAddress  string
	SettlementSigner string
	FeeBasisPoints   uint16
	FeeVersion       uint64
}

func (s *Store) PrepareReceipt(ctx context.Context, requestID string, domain ReceiptDomain) (v1.Receipt, string, string, error) {
	if domain.ChainID == 0 || !common.IsHexAddress(domain.ContractAddress) || !common.IsHexAddress(domain.SettlementSigner) || domain.FeeVersion == 0 || domain.FeeBasisPoints > v1.MaximumFeeBasisPoints {
		return v1.Receipt{}, "", "", v1.ErrInvalidReceipt
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return v1.Receipt{}, "", "", err
	}
	defer tx.Rollback()
	var proposal ReceiptProposal
	var customer, provider, signer, maximum string
	var inputHash, outputHash []byte
	err = tx.QueryRowContext(ctx, `SELECT rp.request_id,rp.session_id,rp.machine_id,rp.offer_id,rp.model,rp.price_version,rp.input_tokens,rp.output_tokens,rp.compute_milliseconds,rp.input_hash,rp.output_hash,rp.completed_at,c.wallet_address,p.wallet_address,m.signer_address,r.maximum_spend::text
		FROM receipt_proposals rp JOIN requests r ON r.id=rp.request_id JOIN sessions se ON se.id=rp.session_id JOIN accounts c ON c.id=se.account_id JOIN machines m ON m.id=rp.machine_id JOIN accounts p ON p.id=m.account_id
		WHERE rp.request_id=$1 AND r.state='completed' AND m.revoked_at IS NULL FOR UPDATE OF r`, requestID).Scan(&proposal.RequestID, &proposal.SessionID, &proposal.MachineID, &proposal.OfferID, &proposal.Model, &proposal.PriceVersion, &proposal.InputTokens, &proposal.OutputTokens, &proposal.ComputeMilliseconds, &inputHash, &outputHash, &proposal.CompletedAt, &customer, &provider, &signer, &maximum)
	if err != nil {
		return v1.Receipt{}, "", "", err
	}
	var offerHashText, modelHashText, capabilityHashText string
	if err := tx.QueryRowContext(ctx, `SELECT offer_hash,model_hash,capability_hash FROM requests WHERE id=$1`, requestID).Scan(&offerHashText, &modelHashText, &capabilityHashText); err != nil {
		return v1.Receipt{}, "", "", err
	}
	parseHash := func(value string) (common.Hash, bool) {
		if len(value) != 66 || !strings.HasPrefix(value, "0x") {
			return common.Hash{}, false
		}
		decoded, err := hex.DecodeString(value[2:])
		return common.BytesToHash(decoded), err == nil && len(decoded) == 32
	}
	offerID, validOffer := parseHash(offerHashText)
	modelHash, validModel := parseHash(modelHashText)
	capabilityHash, validCapability := parseHash(capabilityHashText)
	if !validOffer || !validModel || !validCapability {
		return v1.Receipt{}, "", "", v1.ErrInvalidReceipt
	}
	var inputRate, outputRate, computeRate string
	err = tx.QueryRowContext(ctx, `SELECT input_per_million::text,output_per_million::text,compute_per_second::text FROM chain_offers WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3) AND offer_id=$4 AND version=$5 AND model_hash=$6 AND capability_hash=$7`, domain.ChainID, domain.ContractAddress, provider, offerID.Hex(), proposal.PriceVersion, modelHash.Hex(), capabilityHash.Hex()).Scan(&inputRate, &outputRate, &computeRate)
	if err != nil {
		return v1.Receipt{}, "", "", err
	}
	total, err := exactCharge(proposal.InputTokens, proposal.OutputTokens, proposal.ComputeMilliseconds, inputRate, outputRate, computeRate)
	if err != nil {
		return v1.Receipt{}, "", "", err
	}
	maximumValue, ok := new(big.Int).SetString(maximum, 10)
	if !ok || !maximumValue.IsUint64() || total > maximumValue.Uint64() {
		return v1.Receipt{}, "", "", v1.ErrMaximumExceeded
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, proposal.SessionID); err != nil {
		return v1.Receipt{}, "", "", err
	}
	result, err := tx.ExecContext(ctx, `UPDATE inference_reservations SET amount=$2,released_at=CASE WHEN $2::numeric=0 THEN COALESCE(released_at,now()) ELSE NULL END WHERE request_id=$1 AND amount>=$2`, requestID, decimal(total))
	if err != nil {
		return v1.Receipt{}, "", "", err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return v1.Receipt{}, "", "", v1.ErrMaximumExceeded
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, strings.ToLower(provider)); err != nil {
		return v1.Receipt{}, "", "", err
	}
	var nonce uint64
	err = tx.QueryRowContext(ctx, `SELECT nonce FROM receipt_nonces WHERE request_id=$1`, requestID).Scan(&nonce)
	if errors.Is(err, sql.ErrNoRows) {
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(max(nonce),0)+1 FROM receipt_nonces WHERE lower(provider_address)=lower($1)`, provider).Scan(&nonce); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO receipt_nonces(provider_address,nonce,request_id) VALUES ($1,$2,$3)`, provider, nonce, requestID)
		}
	}
	if err != nil {
		return v1.Receipt{}, "", "", err
	}
	requestHash, sessionHash := common.HexToHash(requestID), common.HexToHash(proposal.SessionID)
	if requestHash == (common.Hash{}) || sessionHash == (common.Hash{}) || len(inputHash) != 32 || len(outputHash) != 32 {
		return v1.Receipt{}, "", "", v1.ErrInvalidReceipt
	}
	receipt := v1.Receipt{
		RequestID: v1.Hash(requestHash), SessionID: v1.Hash(sessionHash), Customer: v1.Address(common.HexToAddress(customer)), Provider: v1.Address(common.HexToAddress(provider)), SettlementSigner: v1.Address(common.HexToAddress(domain.SettlementSigner)), OfferID: v1.Hash(offerID), PriceVersion: proposal.PriceVersion, ModelHash: v1.Hash(modelHash), CapabilityHash: v1.Hash(capabilityHash), InputTokens: proposal.InputTokens, OutputTokens: proposal.OutputTokens, ComputeMilliseconds: proposal.ComputeMilliseconds, MaximumCharge: maximumValue.Uint64(), TotalCharge: total, FeeBasisPoints: domain.FeeBasisPoints, FeeVersion: domain.FeeVersion, Status: v1.ReceiptStatusCompleted, CompletedAt: uint64(proposal.CompletedAt.Unix()), InputHash: v1.Hash(common.BytesToHash(inputHash)), OutputHash: v1.Hash(common.BytesToHash(outputHash)), Nonce: nonce,
	}
	if err := receipt.Validate(); err != nil {
		return v1.Receipt{}, "", "", err
	}
	if err := tx.Commit(); err != nil {
		return v1.Receipt{}, "", "", err
	}
	return receipt, proposal.MachineID, signer, nil
}

func exactCharge(inputTokens, outputTokens, milliseconds uint64, inputRate, outputRate, computeRate string) (uint64, error) {
	total := new(big.Int)
	for _, item := range []struct {
		units       uint64
		rate        string
		denominator uint64
	}{{inputTokens, inputRate, 1_000_000}, {outputTokens, outputRate, 1_000_000}, {milliseconds, computeRate, 1_000}} {
		rate, ok := new(big.Int).SetString(item.rate, 10)
		if !ok || rate.Sign() < 0 {
			return 0, v1.ErrInvalidReceipt
		}
		part := new(big.Int).Mul(new(big.Int).SetUint64(item.units), rate)
		if part.Sign() > 0 {
			part.Add(part, new(big.Int).SetUint64(item.denominator-1))
		}
		part.Div(part, new(big.Int).SetUint64(item.denominator))
		total.Add(total, part)
	}
	if !total.IsUint64() {
		return 0, v1.ErrInvalidReceipt
	}
	return total.Uint64(), nil
}
