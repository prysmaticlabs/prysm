package synccommittee

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

var _ = Pool(&Store{})

// Pool defines the necessary methods for Prysm sync pool to serve
// validators. In the current design, aggregated attestations
// are used by proposers and sync committee messages are used by
// sync aggregators.
type Pool interface {
	// Methods for Sync Contributions.
	SaveSyncCommitteeContribution(sig *ethpb.SyncCommitteeContribution) error
	DeleteSyncCommitteeContributions(slot types.Slot)
	SyncCommitteeContributions(slot types.Slot) []*ethpb.SyncCommitteeContribution

	// Methods for Sync Committee Messages.
	SaveSyncCommitteeSignature(sig *ethpb.SyncCommitteeMessage) error
	DeleteSyncCommitteeSignatures(slot types.Slot)
	SyncCommitteeSignatures(slot types.Slot) []*ethpb.SyncCommitteeMessage
}

// NewPool returns the sync committee store fulfilling the pool interface.
func NewPool() Pool {
	return NewStore()
}
