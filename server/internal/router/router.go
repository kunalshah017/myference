package router

import (
	"errors"
	"sort"
	"strings"
)

var ErrNoEligibleProvider = errors.New("no eligible provider")

type Request struct {
	Model          string
	Capabilities   []string
	MaximumSpend   uint64
	SessionBalance uint64
	PinMachineID   string
}

type Candidate struct {
	MachineID           string
	OfferID             string
	Model               string
	Capabilities        []string
	ConfirmedBond       bool
	Healthy             bool
	Capacity            uint32
	MaximumCost         uint64
	PriceVersion        uint64
	LatencyMilliseconds uint64
	SuccessBasisPoints  uint16
	Reputation          uint64
}

func Select(request Request, candidates []Candidate) (Candidate, error) {
	if strings.TrimSpace(request.Model) == "" || request.MaximumSpend == 0 || request.SessionBalance == 0 {
		return Candidate{}, ErrNoEligibleProvider
	}
	eligible := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if request.PinMachineID != "" && candidate.MachineID != request.PinMachineID {
			continue
		}
		if candidate.MachineID == "" || candidate.OfferID == "" || candidate.Model != request.Model || !candidate.ConfirmedBond || !candidate.Healthy || candidate.Capacity == 0 || candidate.MaximumCost > request.MaximumSpend || candidate.MaximumCost > request.SessionBalance || !containsAll(candidate.Capabilities, request.Capabilities) {
			continue
		}
		eligible = append(eligible, candidate)
	}
	if len(eligible) == 0 {
		return Candidate{}, ErrNoEligibleProvider
	}
	sort.Slice(eligible, func(i, j int) bool {
		left, right := eligible[i], eligible[j]
		if left.MaximumCost != right.MaximumCost {
			return left.MaximumCost < right.MaximumCost
		}
		if left.LatencyMilliseconds != right.LatencyMilliseconds {
			return left.LatencyMilliseconds < right.LatencyMilliseconds
		}
		if left.SuccessBasisPoints != right.SuccessBasisPoints {
			return left.SuccessBasisPoints > right.SuccessBasisPoints
		}
		if left.Reputation != right.Reputation {
			return left.Reputation > right.Reputation
		}
		return left.MachineID < right.MachineID
	})
	return eligible[0], nil
}

func CanRetry(outputStarted bool) bool { return !outputStarted }

func containsAll(have, required []string) bool {
	for _, wanted := range required {
		found := false
		for _, capability := range have {
			if capability == wanted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
