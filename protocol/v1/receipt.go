package v1

import "errors"

const MaximumFeeBasisPoints uint16 = 1_500

var ErrInvalidReceipt = errors.New("invalid usage receipt")

type Hash [32]byte

func (h Hash) IsZero() bool {
	return h == Hash{}
}

type Address [20]byte

func (a Address) IsZero() bool {
	return a == Address{}
}

type ReceiptStatus uint8

const ReceiptStatusCompleted ReceiptStatus = 1

type Receipt struct {
	RequestID           Hash          `json:"request_id"`
	SessionID           Hash          `json:"session_id"`
	Customer            Address       `json:"customer"`
	Provider            Address       `json:"provider"`
	SettlementSigner    Address       `json:"settlement_signer"`
	OfferID             Hash          `json:"offer_id"`
	PriceVersion        uint64        `json:"price_version"`
	ModelHash           Hash          `json:"model_hash"`
	CapabilityHash      Hash          `json:"capability_hash"`
	InputTokens         uint64        `json:"input_tokens"`
	OutputTokens        uint64        `json:"output_tokens"`
	ComputeMilliseconds uint64        `json:"compute_milliseconds"`
	MaximumCharge       uint64        `json:"maximum_charge"`
	TotalCharge         uint64        `json:"total_charge"`
	FeeBasisPoints      uint16        `json:"fee_basis_points"`
	FeeVersion          uint64        `json:"fee_version"`
	Status              ReceiptStatus `json:"status"`
	CompletedAt         uint64        `json:"completed_at"`
	InputHash           Hash          `json:"input_hash"`
	OutputHash          Hash          `json:"output_hash"`
	Nonce               uint64        `json:"nonce"`
}

func (r Receipt) Validate() error {
	if r.RequestID.IsZero() || r.SessionID.IsZero() || r.Customer.IsZero() || r.Provider.IsZero() || r.SettlementSigner.IsZero() || r.OfferID.IsZero() || r.ModelHash.IsZero() || r.CapabilityHash.IsZero() || r.InputHash.IsZero() || r.OutputHash.IsZero() {
		return ErrInvalidReceipt
	}
	if r.PriceVersion == 0 || r.FeeVersion == 0 || r.Status != ReceiptStatusCompleted || r.CompletedAt == 0 || r.Nonce == 0 || r.FeeBasisPoints > MaximumFeeBasisPoints {
		return ErrInvalidReceipt
	}
	if r.TotalCharge > r.MaximumCharge {
		return ErrMaximumExceeded
	}
	return nil
}
