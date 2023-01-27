//go:build fuzz

package cache

import (
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
)

// FakeSyncCommitteeCache is a fake `SyncCommitteeCache` to satisfy fuzzing.
type FakeSyncCommitteeCache struct {
}

// NewSyncCommittee initializes and returns a new SyncCommitteeCache.
func NewSyncCommittee() *FakeSyncCommitteeCache {
	return &FakeSyncCommitteeCache{}
}

// CurrentEpochIndexPosition -- fake.
func (s *FakeSyncCommitteeCache) CurrentPeriodIndexPosition(root [32]byte, valIdx primitives.ValidatorIndex) ([]primitives.CommitteeIndex, error) {
	return nil, nil
}

// NextEpochIndexPosition -- fake.
func (s *FakeSyncCommitteeCache) NextPeriodIndexPosition(root [32]byte, valIdx primitives.ValidatorIndex) ([]primitives.CommitteeIndex, error) {
	return nil, nil
}

// UpdatePositionsInCommittee -- fake.
func (s *FakeSyncCommitteeCache) UpdatePositionsInCommittee(syncCommitteeBoundaryRoot [32]byte, state types.BeaconState) error {
	return nil
}
