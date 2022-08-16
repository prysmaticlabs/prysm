package cache

import (
	"errors"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// ErrNotProposerIndices will be returned when a cache object is not a pointer to
// a ProposerIndices struct.
var ErrNotProposerIndices = errors.New("object is not a proposer indices struct")

// ProposerIndices defines the cached struct for proposer indices.
type ProposerIndices struct {
	BlockRoot       [32]byte
	ProposerIndices []types.ValidatorIndex
}
