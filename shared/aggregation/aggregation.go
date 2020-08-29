// Package aggregation contains implementations of bitlist aggregation algorithms and heuristics.
package aggregation

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/sirupsen/logrus"
)

var _ = logrus.WithField("prefix", "aggregation")

var (
	// ErrBitsOverlap is returned when two bitlists overlap with each other.
	ErrBitsOverlap = errors.New("overlapping aggregation bits")

	// ErrBitsDifferentLen is returned when two bitlists have different lengths.
	ErrBitsDifferentLen = errors.New("different bitlist lengths")

	// ErrInvalidStrategy is returned when invalid aggregation strategy is selected.
	ErrInvalidStrategy = errors.New("invalid aggregation strategy")
)

// Aggregation defines the bitlist aggregation problem solution.
type Aggregation struct {
	// Coverage represents the best available solution to bitlist aggregation.
	Coverage bitfield.Bitlist
	// Keys is a list of candidate keys in the best available solution.
	Keys []int
}

// String provides string representation of a solution.
func (ag *Aggregation) String() string {
	return fmt.Sprintf("{coverage: %#b, keys: %v}", ag.Coverage, ag.Keys)
}
