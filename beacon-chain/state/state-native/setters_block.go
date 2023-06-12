package state_native

import (
	"fmt"

	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// SetLatestBlockHeader in the beacon state.
func (b *BeaconState) SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.latestBlockHeader = ethpb.CopyBeaconBlockHeader(val)
	b.markFieldAsDirty(types.LatestBlockHeader)
	return nil
}

// SetBlockRoots for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBlockRoots(val [][]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.BlockRoots].MinusRef()
	b.sharedFieldReferences[types.BlockRoots] = stateutil.NewRef(1)

	var rootsArr [fieldparams.BlockRootsLength][32]byte
	for i := 0; i < len(rootsArr); i++ {
		copy(rootsArr[i][:], val[i])
	}
	roots := customtypes.BlockRoots(rootsArr)
	b.blockRoots = &roots
	b.markFieldAsDirty(types.BlockRoots)
	b.rebuildTrie[types.BlockRoots] = true
	return nil
}

// UpdateBlockRootAtIndex for the beacon state. Updates the block root
// at a specific index to a new value.
func (b *BeaconState) UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error {
	if uint64(len(b.blockRoots)) <= idx {
		return fmt.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	r := b.blockRoots
	if ref := b.sharedFieldReferences[types.BlockRoots]; ref.Refs() > 1 {
		// Copy elements in underlying array by reference.
		roots := *b.blockRoots
		rootsCopy := roots
		r = &rootsCopy
		ref.MinusRef()
		b.sharedFieldReferences[types.BlockRoots] = stateutil.NewRef(1)
	}

	r[idx] = blockRoot
	b.blockRoots = r

	b.markFieldAsDirty(types.BlockRoots)
	b.addDirtyIndices(types.BlockRoots, []uint64{idx})
	return nil
}
