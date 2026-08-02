package v1

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestReceiptSigningRecoversMachineSignerAndBindsDomain(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	copyAddress := func(raw []byte) Address { var value Address; copy(value[:], raw); return value }
	hash := func(seed byte) Hash { var value Hash; value[0] = seed; return value }
	receipt := Receipt{
		RequestID: hash(1), SessionID: hash(2), Customer: copyAddress(make([]byte, 20)), Provider: copyAddress(crypto.PubkeyToAddress(key.PublicKey).Bytes()), SettlementSigner: copyAddress(append(make([]byte, 19), 2)),
		OfferID: hash(3), PriceVersion: 1, ModelHash: hash(4), CapabilityHash: hash(5), InputTokens: 2, OutputTokens: 3, ComputeMilliseconds: 4, MaximumCharge: 10, TotalCharge: 9, FeeBasisPoints: 500, FeeVersion: 1, Status: ReceiptStatusCompleted, CompletedAt: 100, InputHash: hash(6), OutputHash: hash(7), Nonce: 1,
	}
	receipt.Customer[19] = 1
	contract := copyAddress(append(make([]byte, 19), 3))
	signature, err := SignReceipt(receipt, 10143, contract, key)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := RecoverReceiptSigner(receipt, 10143, contract, signature)
	if err != nil || recovered != receipt.Provider {
		t.Fatalf("recovered=%x want=%x err=%v", recovered, receipt.Provider, err)
	}
	other, err := RecoverReceiptSigner(receipt, 1, contract, signature)
	if err != nil || other == receipt.Provider {
		t.Fatal("signature unexpectedly recovered the provider on another chain")
	}
}
