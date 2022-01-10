// +build libfuzzer

package cache

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native"
)

// FakeSyncCommitteeCache is a fake `SyncCommitteeCache` to satisfy fuzzing.
type FakeSyncCommitteeCache struct {
}

// NewSyncCommittee initializes and returns a new SyncCommitteeCache.
func NewSyncCommittee() *FakeSyncCommitteeCache {
	return &FakeSyncCommitteeCache{}
}

// CurrentEpochIndexPosition -- fake.
func (s *FakeSyncCommitteeCache) CurrentPeriodIndexPosition(root [32]byte, valIdx types.ValidatorIndex) ([]types.CommitteeIndex, error) {
	return nil, nil
}

// NextEpochIndexPosition -- fake.
func (s *FakeSyncCommitteeCache) NextPeriodIndexPosition(root [32]byte, valIdx types.ValidatorIndex) ([]types.CommitteeIndex, error) {
	return nil, nil
}

// UpdatePositionsInCommittee -- fake.
func (s *FakeSyncCommitteeCache) UpdatePositionsInCommittee(syncCommitteeBoundaryRoot [32]byte, state state.BeaconStateAltair) error {
	return nil
}
