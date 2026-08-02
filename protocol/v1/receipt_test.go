package v1

import (
	"errors"
	"testing"
)

func TestReceiptValidatesHashesAddressesAndAmounts(t *testing.T) {
	receipt := validReceipt()
	if err := receipt.Validate(); err != nil {
		t.Fatal(err)
	}

	receipt.Provider = Address{}
	if err := receipt.Validate(); !errors.Is(err, ErrInvalidReceipt) {
		t.Fatalf("expected ErrInvalidReceipt for zero provider, got %v", err)
	}

	receipt = validReceipt()
	receipt.TotalCharge = receipt.MaximumCharge + 1
	if err := receipt.Validate(); !errors.Is(err, ErrMaximumExceeded) {
		t.Fatalf("expected ErrMaximumExceeded, got %v", err)
	}
}

func TestReceiptRejectsZeroIdentityFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Receipt)
	}{
		{"request id", func(r *Receipt) { r.RequestID = Hash{} }},
		{"session id", func(r *Receipt) { r.SessionID = Hash{} }},
		{"customer", func(r *Receipt) { r.Customer = Address{} }},
		{"settlement signer", func(r *Receipt) { r.SettlementSigner = Address{} }},
		{"offer id", func(r *Receipt) { r.OfferID = Hash{} }},
		{"model hash", func(r *Receipt) { r.ModelHash = Hash{} }},
		{"capability hash", func(r *Receipt) { r.CapabilityHash = Hash{} }},
		{"input hash", func(r *Receipt) { r.InputHash = Hash{} }},
		{"output hash", func(r *Receipt) { r.OutputHash = Hash{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			receipt := validReceipt()
			test.mutate(&receipt)
			if err := receipt.Validate(); !errors.Is(err, ErrInvalidReceipt) {
				t.Fatalf("expected ErrInvalidReceipt, got %v", err)
			}
		})
	}
}

func validReceipt() Receipt {
	return Receipt{
		RequestID:           hashWithLastByte(1),
		SessionID:           hashWithLastByte(2),
		Customer:            addressWithLastByte(1),
		Provider:            addressWithLastByte(2),
		SettlementSigner:    addressWithLastByte(3),
		OfferID:             hashWithLastByte(3),
		PriceVersion:        1,
		ModelHash:           hashWithLastByte(4),
		CapabilityHash:      hashWithLastByte(5),
		InputTokens:         10,
		OutputTokens:        20,
		ComputeMilliseconds: 30,
		MaximumCharge:       1_000,
		TotalCharge:         100,
		FeeBasisPoints:      500,
		FeeVersion:          1,
		Status:              ReceiptStatusCompleted,
		CompletedAt:         1_754_118_400,
		InputHash:           hashWithLastByte(6),
		OutputHash:          hashWithLastByte(7),
		Nonce:               1,
	}
}

func hashWithLastByte(value byte) Hash {
	var hash Hash
	hash[len(hash)-1] = value
	return hash
}

func addressWithLastByte(value byte) Address {
	var address Address
	address[len(address)-1] = value
	return address
}
