package state_native

import (
	"github.com/pkg/errors"
	customtypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/custom-types"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
)

// SetStateRoots for the beacon state. Updates the state roots
// to a new value by overwriting the previous value.
func (b *BeaconState) SetStateRoots(val [][]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[nativetypes.StateRoots].MinusRef()
	b.sharedFieldReferences[nativetypes.StateRoots] = stateutil.NewRef(1)

	var rootsArr [fieldparams.StateRootsLength][32]byte
	for i := 0; i < len(rootsArr); i++ {
		copy(rootsArr[i][:], val[i])
	}
	roots := customtypes.StateRoots(rootsArr)
	b.stateRoots = &roots
	b.markFieldAsDirty(nativetypes.StateRoots)
	b.rebuildTrie[nativetypes.StateRoots] = true
	return nil
}

// UpdateStateRootAtIndex for the beacon state. Updates the state root
// at a specific index to a new value.
func (b *BeaconState) UpdateStateRootAtIndex(idx uint64, stateRoot [32]byte) error {
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
	if ref := b.sharedFieldReferences[nativetypes.StateRoots]; ref.Refs() > 1 {
		// Copy elements in underlying array by reference.
		roots := *b.stateRoots
		rootsCopy := roots
		r = &rootsCopy
		ref.MinusRef()
		b.sharedFieldReferences[nativetypes.StateRoots] = stateutil.NewRef(1)
	}

	r[idx] = stateRoot
	b.stateRoots = r

	b.markFieldAsDirty(nativetypes.StateRoots)
	b.addDirtyIndices(nativetypes.StateRoots, []uint64{idx})
	return nil
}
