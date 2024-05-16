package electra

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// sortableIndices implements the Sort interface to sort newly activated validator indices
// by activation epoch and by index number.
type sortableIndices struct {
	indices    []primitives.ValidatorIndex
	validators []*ethpb.Validator
}

// Len is the number of elements in the collection.
func (s sortableIndices) Len() int { return len(s.indices) }

// Swap swaps the elements with indexes i and j.
func (s sortableIndices) Swap(i, j int) { s.indices[i], s.indices[j] = s.indices[j], s.indices[i] }

// Less reports whether the element with index i must sort before the element with index j.
func (s sortableIndices) Less(i, j int) bool {
	if s.validators[s.indices[i]].ActivationEligibilityEpoch == s.validators[s.indices[j]].ActivationEligibilityEpoch {
		return s.indices[i] < s.indices[j]
	}
	return s.validators[s.indices[i]].ActivationEligibilityEpoch < s.validators[s.indices[j]].ActivationEligibilityEpoch
}
