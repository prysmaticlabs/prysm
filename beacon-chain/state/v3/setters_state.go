package v3

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
)

// SetStateRoots for the beacon state. Updates the state roots
// to a new value by overwriting the previous value.
func (b *BeaconState) SetStateRoots(val [][]byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[stateRoots].MinusRef()
	b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)

	b.state.StateRoots = val
	b.markFieldAsDirty(stateRoots)
	b.rebuildTrie[stateRoots] = true
	return nil
}

// UpdateStateRootAtIndex for the beacon state. Updates the state root
// at a specific index to a new value.
func (b *BeaconState) UpdateStateRootAtIndex(idx uint64, stateRoot [32]byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}

	b.lock.RLock()
	if uint64(len(b.state.StateRoots)) <= idx {
		b.lock.RUnlock()
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if we hold the only reference to the shared state roots slice.
	r := b.state.StateRoots
	if ref := b.sharedFieldReferences[stateRoots]; ref.Refs() > 1 {
		// Copy elements in underlying array by reference.
		r = make([][]byte, len(b.state.StateRoots))
		copy(r, b.state.StateRoots)
		ref.MinusRef()
		b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)
	}

	r[idx] = stateRoot[:]
	b.state.StateRoots = r

	b.markFieldAsDirty(stateRoots)
	b.addDirtyIndices(stateRoots, []uint64{idx})
	return nil
}
