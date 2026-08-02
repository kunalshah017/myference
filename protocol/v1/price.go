package v1

import (
	"errors"
	"math/bits"
)

const (
	tokensPerMillion = uint64(1_000_000)
	millisPerSecond  = uint64(1_000)
)

var (
	ErrOverflow        = errors.New("price calculation overflow")
	ErrMaximumExceeded = errors.New("maximum charge exceeded")
)

type Price struct {
	InputPerMillion  uint64 `json:"input_per_million"`
	OutputPerMillion uint64 `json:"output_per_million"`
	ComputePerSecond uint64 `json:"compute_per_second"`
}

func (p Price) Charge(inputTokens, outputTokens, computeMillis, maxWei uint64) (uint64, error) {
	input, err := multiplyDivideCeil(inputTokens, p.InputPerMillion, tokensPerMillion)
	if err != nil {
		return 0, err
	}
	output, err := multiplyDivideCeil(outputTokens, p.OutputPerMillion, tokensPerMillion)
	if err != nil {
		return 0, err
	}
	compute, err := multiplyDivideCeil(computeMillis, p.ComputePerSecond, millisPerSecond)
	if err != nil {
		return 0, err
	}

	total, carry := bits.Add64(input, output, 0)
	if carry != 0 {
		return 0, ErrOverflow
	}
	total, carry = bits.Add64(total, compute, 0)
	if carry != 0 {
		return 0, ErrOverflow
	}
	if total > maxWei {
		return 0, ErrMaximumExceeded
	}
	return total, nil
}

func multiplyDivideCeil(units, rate, denominator uint64) (uint64, error) {
	if units == 0 || rate == 0 {
		return 0, nil
	}
	hi, lo := bits.Mul64(units, rate)
	if hi >= denominator {
		return 0, ErrOverflow
	}
	quotient, remainder := bits.Div64(hi, lo, denominator)
	if remainder == 0 {
		return quotient, nil
	}
	quotient, carry := bits.Add64(quotient, 1, 0)
	if carry != 0 {
		return 0, ErrOverflow
	}
	return quotient, nil
}
