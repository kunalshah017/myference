package chain

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type deployedCodeReader struct{ deployment uint64 }

func (r deployedCodeReader) CodeAt(_ context.Context, _ common.Address, block *big.Int) ([]byte, error) {
	if block.Uint64() >= r.deployment {
		return []byte{1}, nil
	}
	return nil, nil
}

func TestBatchEndCapsRangeWithoutOverflow(t *testing.T) {
	tests := []struct {
		name                     string
		next, target, size, want uint64
	}{
		{name: "full batch", next: 100, target: 999, size: 250, want: 349},
		{name: "final partial batch", next: 900, target: 999, size: 250, want: 999},
		{name: "single block", next: 42, target: 42, size: 250, want: 42},
		{name: "overflow safe", next: ^uint64(0) - 2, target: ^uint64(0), size: 250, want: ^uint64(0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := batchEnd(test.next, test.target, test.size); got != test.want {
				t.Fatalf("batchEnd(%d, %d, %d)=%d want %d", test.next, test.target, test.size, got, test.want)
			}
		})
	}
}

func TestIndexerBatchFitsMonadPublicRPCLimit(t *testing.T) {
	if indexBatchSize > 100 {
		t.Fatalf("index batch size %d exceeds Monad public RPC limit", indexBatchSize)
	}
}

func TestFirstCodeBlockSkipsPreDeploymentHistory(t *testing.T) {
	reader := deployedCodeReader{deployment: 50_000_000}
	got, err := firstCodeBlock(context.Background(), reader, common.Address{1}, 0, 50_378_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != reader.deployment {
		t.Fatalf("firstCodeBlock=%d want %d", got, reader.deployment)
	}
}

func TestFirstCodeBlockRespectsConfiguredPostDeploymentStart(t *testing.T) {
	reader := deployedCodeReader{deployment: 50_000_000}
	got, err := firstCodeBlock(context.Background(), reader, common.Address{1}, 50_100_000, 50_378_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 50_100_000 {
		t.Fatalf("firstCodeBlock=%d want configured start", got)
	}
}
