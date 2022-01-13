package v3

import (
	"fmt"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
)

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if b.randaoMixes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	mixesArr := b.randaoMixesInternal()
	mixes := make([][]byte, len(mixesArr))
	for i, m := range mixesArr {
		tmp := m
		mixes[i] = tmp[:]
	}

	return mixes
}

// randaoMixesInternal of block proposers on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixesInternal() *customtypes.RandaoMixes {
	return b.randaoMixes
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([]byte, error) {
	if b.randaoMixes == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	m, err := b.randaoMixAtIndex(idx)
	if err != nil {
		return nil, err
	}
	return m[:], nil
}

// randaoMixAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixAtIndex(idx uint64) ([32]byte, error) {
	if uint64(len(b.randaoMixes)) <= idx {
		return [32]byte{}, fmt.Errorf("index %d out of range", idx)
	}

	return b.randaoMixes[idx], nil
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	if b.randaoMixes == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixesLength()
}

// randaoMixesLength returns the length of the randao mixes slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixesLength() int {
	if b.randaoMixes == nil {
		return 0
	}

	return len(b.randaoMixes)
}
