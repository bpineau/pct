// Package percent provides elementary percentage arithmetic.
package percent

import (
	"errors"
	"math"
)

// ErrZeroReference is returned when an operation needs a non-zero reference
// value, such as the starting point of a percentage change.
var ErrZeroReference = errors.New("reference value is zero")

// Add returns n increased by p percent. A negative p decreases n.
func Add(p, n float64) float64 {
	return n * (1 + p/100)
}

// Change returns the percentage that, added to from, yields to.
// It returns ErrZeroReference if from is zero: no percentage of zero can
// reach another value.
func Change(from, to float64) (float64, error) {
	if from == 0 {
		return 0, ErrZeroReference
	}
	return (to/from - 1) * 100, nil
}

// Share returns the percentage that part represents of whole.
// It returns ErrZeroReference if whole is zero.
func Share(part, whole float64) (float64, error) {
	if whole == 0 {
		return 0, ErrZeroReference
	}
	return part / whole * 100, nil
}

// Compound returns n increased by p percent, compounded over the given
// number of periods.
func Compound(p, n float64, periods int) float64 {
	return n * math.Pow(1+p/100, float64(periods))
}
