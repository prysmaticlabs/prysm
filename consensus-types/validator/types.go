package validator

import (
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type SyncCommitteeSubscription struct {
	ValidatorIndex       primitives.ValidatorIndex
	SyncCommitteeIndices []uint64
	UntilEpoch           primitives.Epoch
}
