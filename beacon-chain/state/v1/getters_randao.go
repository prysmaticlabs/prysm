package v1

import (
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.RandaoMixes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixes()
}

// randaoMixes of block proposers on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixes() [][]byte {
	if !b.hasInnerState() {
		return nil
	}

	return bytesutil.SafeCopy2dBytes(b.state.RandaoMixes)
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.RandaoMixes == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixAtIndex(idx)
}

// randaoMixAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) randaoMixAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}

	return bytesutil.SafeCopyRootAtIndex(b.state.RandaoMixes, idx)
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.RandaoMixes == nil {
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
	if b.state.RandaoMixes == nil {
		return 0
	}

	return len(b.state.RandaoMixes)
}
