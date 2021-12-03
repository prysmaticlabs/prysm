package v3

import (
	"github.com/pkg/errors"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
)

// SetStateRoots for the beacon state. Updates the state roots
// to a new value by overwriting the previous value.
func (b *BeaconState) SetStateRoots(val *[8192][32]byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[stateRoots].MinusRef()
	b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)

	roots := customtypes.StateRoots(*val)
	b.stateRoots = &roots
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
	if uint64(len(b.stateRoots)) <= idx {
		b.lock.RUnlock()
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if we hold the only reference to the shared state roots slice.
	r := b.stateRoots
	if ref := b.sharedFieldReferences[stateRoots]; ref.Refs() > 1 {
		// Copy elements in underlying array by reference.
		roots := *b.stateRoots
		rootsCopy := roots
		r = &rootsCopy
		ref.MinusRef()
		b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)
	}

	r[idx] = stateRoot
	b.stateRoots = r

	b.markFieldAsDirty(stateRoots)
	b.addDirtyIndices(stateRoots, []uint64{idx})
	return nil
}
