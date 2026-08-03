package router

import (
	"errors"
	"math/big"
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
	InputPerMillion     string
	OutputPerMillion    string
	ComputePerSecond    string
}

func WorstCaseCost(candidate Candidate, maximumInputTokens, maximumOutputTokens, maximumComputeMilliseconds uint64) (uint64, error) {
	total := new(big.Int)
	for _, item := range []struct {
		units   uint64
		rate    string
		divisor uint64
	}{{maximumInputTokens, candidate.InputPerMillion, 1_000_000}, {maximumOutputTokens, candidate.OutputPerMillion, 1_000_000}, {maximumComputeMilliseconds, candidate.ComputePerSecond, 1_000}} {
		rate, ok := parseRate(item.rate)
		if !ok {
			return 0, ErrNoEligibleProvider
		}
		part := new(big.Int).Mul(new(big.Int).SetUint64(item.units), rate)
		part.Add(part, new(big.Int).SetUint64(item.divisor-1)).Div(part, new(big.Int).SetUint64(item.divisor))
		total.Add(total, part)
	}
	if !total.IsUint64() {
		return 0, ErrNoEligibleProvider
	}
	return total.Uint64(), nil
}

func ValidRate(value string) bool {
	_, ok := parseRate(value)
	return ok
}

func HasPricing(candidate Candidate) bool {
	for _, value := range []string{candidate.InputPerMillion, candidate.OutputPerMillion, candidate.ComputePerSecond} {
		rate, ok := parseRate(value)
		if !ok {
			return false
		}
		if rate.Sign() != 0 {
			return true
		}
	}
	return false
}

func SumRates(input, output, compute string) (string, error) {
	total := new(big.Int)
	for _, value := range []string{input, output, compute} {
		rate, ok := parseRate(value)
		if !ok {
			return "", ErrNoEligibleProvider
		}
		total.Add(total, rate)
	}
	return total.String(), nil
}

func parseRate(value string) (*big.Int, bool) {
	if value == "" || len(value) > 1 && value[0] == '0' {
		return nil, false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return nil, false
		}
	}
	rate, ok := new(big.Int).SetString(value, 10)
	if !ok || rate.BitLen() > 256 {
		return nil, false
	}
	return rate, true
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
		if candidate.MachineID == "" || candidate.OfferID == "" || candidate.Model != request.Model || !candidate.ConfirmedBond || !candidate.Healthy || candidate.Capacity == 0 || candidate.MaximumCost == 0 || candidate.MaximumCost > request.MaximumSpend || candidate.MaximumCost > request.SessionBalance || !containsAll(candidate.Capabilities, request.Capabilities) {
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
