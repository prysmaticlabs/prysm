package cache

import "errors"

// ErrNotCommittee will be returned when a cache object is not a pointer to
// a Committee struct.
var ErrNotCommittee = errors.New("object is not a committee struct")

// Committees defines the shuffled committees seed.
type Committees struct {
	CommitteeCount  uint64
	Seed            [32]byte
	ShuffledIndices []uint64
	SortedIndices   []uint64
}
