package store

import (
	"testing"

	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestRuntimeEvidenceAllowsOnlyVersionedDigestRotation(t *testing.T) {
	sameVersion := v1.OfferCapacity{PriceVersion: 2, EvidenceDigest: "sha256:new"}
	if runtimeEvidenceEligible("sha256:old", 2, sameVersion) {
		t.Fatal("same-version digest drift accepted")
	}
	newVersion := sameVersion
	newVersion.PriceVersion = 3
	if !runtimeEvidenceEligible("sha256:old", 2, newVersion) {
		t.Fatal("new offer version could not rotate digest")
	}
}

func TestComputeOnlyOffersCannotChargeTokenRates(t *testing.T) {
	if meteringRatesEligible("compute_only", "1", "0") {
		t.Fatal("compute-only offer accepted an input-token rate")
	}
	if !meteringRatesEligible("compute_only", "0", "0") || !meteringRatesEligible("tokens_and_compute", "1", "2") {
		t.Fatal("valid metering rates rejected")
	}
}

func TestRoutingRatesAcceptUint256Pricing(t *testing.T) {
	maximum, ok := routingRates("tokens_and_compute", "292000000000000000000", "1752000000000000000000", "4866666666666667")
	if !ok || maximum != "2044004866666666666667" {
		t.Fatalf("maximum=%q ok=%v", maximum, ok)
	}
}
