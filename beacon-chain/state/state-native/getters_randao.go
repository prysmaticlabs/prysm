package state_native

import (
	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
)

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if b.randaoMixes == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	rm := customtypes.RandaoMixes(b.randaoMixes.Value(b))
	rmSlice := rm.Slice()
	rmCopy := make([][]byte, len(rmSlice))
	for i, v := range rmSlice {
		copy(rmCopy[i], v)
	}
	return rmCopy
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([]byte, error) {
	if b.randaoMixes == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	m, err := b.randaoMixes.At(b, idx)
	if err != nil {
		return nil, err
	}
	return m[:], nil
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	if b.randaoMixes == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.randaoMixes.Len(b)
}
