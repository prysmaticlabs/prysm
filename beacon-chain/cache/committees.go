package cache

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

// Committees defines the shuffled committees seed.
type Committees struct {
	CommitteeCount  uint64
	Seed            [32]byte
	ShuffledIndices []primitives.ValidatorIndex
	SortedIndices   []primitives.ValidatorIndex
}
