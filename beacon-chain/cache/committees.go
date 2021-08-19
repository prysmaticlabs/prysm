package cache

import (
	"errors"

	types "github.com/prysmaticlabs/eth2-types"
)

// ErrNotCommittee will be returned when a cache object is not a pointer to
// a Committee struct.
var ErrNotCommittee = errors.New("object is not a committee struct")

// ErrNonCommitteeKey will be returned when the committee key does not exist in cache.
var ErrNonCommitteeKey = errors.New("committee key does not exist")

// Committees defines the shuffled committees seed.
type Committees struct {
	CommitteeCount  uint64
	Seed            [32]byte
	ShuffledIndices []types.ValidatorIndex
	SortedIndices   []types.ValidatorIndex
}
