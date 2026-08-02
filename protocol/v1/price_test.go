package v1

import (
	"errors"
	"testing"
)

func TestChargeRoundsUpAndNeverExceedsMaximum(t *testing.T) {
	price := Price{InputPerMillion: 100, OutputPerMillion: 200, ComputePerSecond: 300}

	charge, err := price.Charge(1, 2, 1, 1_000)
	if err != nil || charge != 3 {
		t.Fatalf("charge=%d err=%v", charge, err)
	}

	charge, err = price.Charge(1_000_000, 1_000_000, 1_000, 600)
	if err != nil || charge != 600 {
		t.Fatalf("full-unit charge=%d err=%v", charge, err)
	}

	if _, err := price.Charge(1_000_000, 1_000_000, 1_000, 1); !errors.Is(err, ErrMaximumExceeded) {
		t.Fatalf("expected ErrMaximumExceeded, got %v", err)
	}
}
