package money

import (
	"errors"
	"math"
)

// Cents хранит деньги целым числом. Например, 1099 означает 10.99.
type Cents int64

var ErrInvalidAmount = errors.New("amount must be finite and have at most two decimal places")

func FromFloat(value float64) (Cents, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, ErrInvalidAmount
	}

	cents := value * 100
	rounded := math.Round(cents)
	if math.Abs(cents-rounded) > 0.000001 {
		return 0, ErrInvalidAmount
	}

	return Cents(rounded), nil
}

func (c Cents) Float64() float64 {
	return float64(c) / 100
}

func (c Cents) Mul(quantity int) Cents {
	return c * Cents(quantity)
}

func Min(a, b Cents) Cents {
	if a < b {
		return a
	}
	return b
}
