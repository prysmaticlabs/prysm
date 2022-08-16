package v2

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
)

// SetRandaoMixes for the beacon state. Updates the entire
// randao mixes to a new value by overwriting the previous one.
func (b *BeaconState) SetRandaoMixes(val [][]byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[randaoMixes].MinusRef()
	b.sharedFieldReferences[randaoMixes] = stateutil.NewRef(1)

	b.state.RandaoMixes = val
	b.markFieldAsDirty(randaoMixes)
	b.rebuildTrie[randaoMixes] = true
	return nil
}

// UpdateRandaoMixesAtIndex for the beacon state. Updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateRandaoMixesAtIndex(idx uint64, val []byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	if uint64(len(b.state.RandaoMixes)) <= idx {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	mixes := b.state.RandaoMixes
	if refs := b.sharedFieldReferences[randaoMixes].Refs(); refs > 1 {
		// Copy elements in underlying array by reference.
		mixes = make([][]byte, len(b.state.RandaoMixes))
		copy(mixes, b.state.RandaoMixes)
		b.sharedFieldReferences[randaoMixes].MinusRef()
		b.sharedFieldReferences[randaoMixes] = stateutil.NewRef(1)
	}

	mixes[idx] = val
	b.state.RandaoMixes = mixes
	b.markFieldAsDirty(randaoMixes)
	b.addDirtyIndices(randaoMixes, []uint64{idx})

	return nil
}
