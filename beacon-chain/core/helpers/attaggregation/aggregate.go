package attaggregation

import "errors"

// AttestationAggregationStrategy defines attestation aggregation strategy.
type AttestationAggregationStrategy string


var (
	// ErrBitsOverlap is returned when two attestations aggregation bits overlap with each other.
	ErrBitsOverlap = errors.New("overlapping aggregation bits")

	// ErrBitsDifferentLen is returned when two attestation aggregation bits have different lengths.
	ErrBitsDifferentLen = errors.New("different bitlist lengths")

	// ErrInvalidStrategy is returned when invalid aggregation strategy is selected.
	ErrInvalidStrategy = errors.New("invalid aggregation strategy")
)
