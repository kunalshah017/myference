package router

import (
	"errors"
	"testing"
)

func TestSelectFiltersEveryEconomicAndRuntimeRequirement(t *testing.T) {
	base := Candidate{
		MachineID: "machine-good", OfferID: "offer", Model: "qwen",
		Capabilities: []string{"text", "stream"}, ConfirmedBond: true, Healthy: true,
		Capacity: 1, MaximumCost: 90, LatencyMilliseconds: 50, SuccessBasisPoints: 9_900, Reputation: 80,
	}
	candidates := []Candidate{
		with(base, func(c *Candidate) { c.MachineID = "no-bond"; c.ConfirmedBond = false }),
		with(base, func(c *Candidate) { c.MachineID = "wrong-model"; c.Model = "other" }),
		with(base, func(c *Candidate) { c.MachineID = "no-capability"; c.Capabilities = []string{"text"} }),
		with(base, func(c *Candidate) { c.MachineID = "unhealthy"; c.Healthy = false }),
		with(base, func(c *Candidate) { c.MachineID = "full"; c.Capacity = 0 }),
		with(base, func(c *Candidate) { c.MachineID = "too-expensive"; c.MaximumCost = 101 }),
		base,
	}
	selected, err := Select(Request{Model: "qwen", Capabilities: []string{"text", "stream"}, MaximumSpend: 100, SessionBalance: 100}, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if selected.MachineID != "machine-good" {
		t.Fatalf("selected %+v", selected)
	}
	if _, err := Select(Request{Model: "qwen", Capabilities: []string{"text", "stream"}, MaximumSpend: 100, SessionBalance: 89}, []Candidate{base}); !errors.Is(err, ErrNoEligibleProvider) {
		t.Fatalf("expected session-balance rejection, got %v", err)
	}
}

func TestSelectRanksStablyAndPinCannotBypassEligibility(t *testing.T) {
	base := Candidate{OfferID: "offer", Model: "qwen", Capabilities: []string{"text"}, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 50, LatencyMilliseconds: 20, SuccessBasisPoints: 9_900, Reputation: 90}
	worsePrice := with(base, func(c *Candidate) { c.MachineID = "price"; c.MaximumCost = 51 })
	worseLatency := with(base, func(c *Candidate) { c.MachineID = "latency"; c.LatencyMilliseconds = 21 })
	worseSuccess := with(base, func(c *Candidate) { c.MachineID = "success"; c.SuccessBasisPoints = 9_800 })
	worseReputation := with(base, func(c *Candidate) { c.MachineID = "reputation"; c.Reputation = 89 })
	best := with(base, func(c *Candidate) { c.MachineID = "best" })
	request := Request{Model: "qwen", Capabilities: []string{"text"}, MaximumSpend: 100, SessionBalance: 100}
	selected, err := Select(request, []Candidate{worsePrice, worseLatency, worseSuccess, worseReputation, best})
	if err != nil || selected.MachineID != "best" {
		t.Fatalf("selected=%+v err=%v", selected, err)
	}
	request.PinMachineID = "price"
	selected, err = Select(request, []Candidate{best, worsePrice})
	if err != nil || selected.MachineID != "price" {
		t.Fatalf("pinned selected=%+v err=%v", selected, err)
	}
	request.PinMachineID = "offline"
	if _, err := Select(request, []Candidate{with(base, func(c *Candidate) { c.MachineID = "offline"; c.Healthy = false })}); !errors.Is(err, ErrNoEligibleProvider) {
		t.Fatalf("pin bypassed eligibility: %v", err)
	}
}

func TestRetryStopsAfterFirstOutput(t *testing.T) {
	if !CanRetry(false) || CanRetry(true) {
		t.Fatal("retry boundary is incorrect")
	}
}

func TestWorstCaseCostUsesRequestedTokensAndComputeDeadline(t *testing.T) {
	cost, err := WorstCaseCost(Candidate{InputPerMillion: 1_000_000, OutputPerMillion: 2_000_000, ComputePerSecond: 3}, 4, 5, 30_000)
	if err != nil || cost != 104 {
		t.Fatalf("cost=%d err=%v", cost, err)
	}
}

func with(candidate Candidate, change func(*Candidate)) Candidate {
	change(&candidate)
	return candidate
}
