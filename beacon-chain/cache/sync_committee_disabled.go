// +build libfuzzer

package cache

import (
	types "github.com/prysmaticlabs/eth2-types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
)

// FakeSyncCommitteeCache is a fake `SyncCommitteeCache` to satisfy fuzzing.
type FakeSyncCommitteeCache struct {
}

// NewSyncCommittee initializes and returns a new SyncCommitteeCache.
func NewSyncCommittee() *FakeSyncCommitteeCache {
	return &FakeSyncCommitteeCache{}
}

// CurrentPeriodIndexPosition -- fake.
func (s *FakeSyncCommitteeCache) CurrentPeriodIndexPosition(root [32]byte, valIdx types.ValidatorIndex) ([]uint64, error) {
	return nil, nil
}

// NextPeriodIndexPosition -- fake.
func (s *FakeSyncCommitteeCache) NextPeriodIndexPosition(root [32]byte, valIdx types.ValidatorIndex) ([]uint64, error) {
	return nil, nil
}

// UpdatePositionsInCommittee -- fake.
func (s *FakeSyncCommitteeCache) UpdatePositionsInCommittee(syncCommitteeBoundaryRoot [32]byte, state iface.BeaconStateAltair) error {
	return nil
}
