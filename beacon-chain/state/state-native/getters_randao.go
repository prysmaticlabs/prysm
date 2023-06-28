package state_native

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/config/features"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	b.lock.RLock()
	defer b.lock.RUnlock()

	mixes := b.randaoMixesVal()
	mixesCopy := make([][]byte, len(mixes))
	for i, m := range mixes {
		mixesCopy[i] = make([]byte, 32)
		copy(mixesCopy[i], m[:])
	}
	return mixesCopy
}

func (b *BeaconState) randaoMixesVal() [][32]byte {
	if features.Get().EnableExperimentalState {
		if b.randaoMixesMultiValue == nil {
			return nil
		}
		return b.randaoMixesMultiValue.Value(b)
	}
	return b.randaoMixes
}

// RandaoMixAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) RandaoMixAtIndex(idx uint64) ([]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if features.Get().EnableExperimentalState {
		if b.randaoMixesMultiValue == nil {
			return nil, nil
		}
		r, err := b.randaoMixesMultiValue.At(b, idx)
		if err != nil {
			return nil, err
		}
		return bytesutil.SafeCopyBytes(r[:]), nil
	}

	if b.randaoMixes == nil {
		return nil, nil
	}
	if uint64(len(b.randaoMixes)) <= idx {
		return []byte{}, fmt.Errorf("index %d out of bounds", idx)
	}
	return bytesutil.SafeCopyBytes(b.randaoMixes[idx][:]), nil
}

// RandaoMixesLength returns the length of the randao mixes slice.
func (b *BeaconState) RandaoMixesLength() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if features.Get().EnableExperimentalState {
		if b.randaoMixesMultiValue == nil {
			return 0
		}
		return b.randaoMixesMultiValue.Len(b)
	}
	if b.randaoMixes == nil {
		return 0
	}
	return len(b.randaoMixes)
}
