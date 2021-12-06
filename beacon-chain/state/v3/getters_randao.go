package v3

import (
	"fmt"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
)

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() *[65536][32]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.randaoMixes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	mixes := [customtypes.RandaoMixesSize][32]byte(*b.randaoMixesInternal())
	return &mixes
}

// randaoMixesInternal of block proposers on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixesInternal() *customtypes.RandaoMixes {
	if !b.hasInnerState() {
		return nil
	}

	return b.randaoMixes
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	if b.randaoMixes == nil {
		return [32]byte{}, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixAtIndex(idx)
}

// randaoMixAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	if uint64(len(b.randaoMixes)) <= idx {
		return [32]byte{}, fmt.Errorf("index %d out of range", idx)
	}

	return b.randaoMixes[idx], nil
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	if !b.hasInnerState() {
		return 0
	}
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
	if !b.hasInnerState() {
		return 0
	}
	if b.randaoMixes == nil {
		return 0
	}

	return len(b.randaoMixes)
}
