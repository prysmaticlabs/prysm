// Package aggregation contains implementations of bit aggregation algorithms and heuristics.
package aggregation

import (
	"errors"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "aggregation")

var (
	// ErrBitsOverlap is returned when two bitlists overlap with each other.
	ErrBitsOverlap = errors.New("overlapping aggregation bits")

	// ErrBitsDifferentLen is returned when two bitlists have different lengths.
	ErrBitsDifferentLen = errors.New("different bitlist lengths")

	// ErrInvalidStrategy is returned when invalid aggregation strategy is selected.
	ErrInvalidStrategy = errors.New("invalid aggregation strategy")
)
