package v1

import (
	"fmt"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetLatestBlockHeader in the beacon state.
func (b *BeaconState) SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.latestBlockHeader = ethpb.CopyBeaconBlockHeader(val)
	b.markFieldAsDirty(latestBlockHeader)
	return nil
}

// SetBlockRoots for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetBlockRoots(val *[8192][32]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[blockRoots].MinusRef()
	b.sharedFieldReferences[blockRoots] = stateutil.NewRef(1)

	roots := customtypes.BlockRoots(*val)
	b.blockRoots = &roots
	b.markFieldAsDirty(blockRoots)
	b.rebuildTrie[blockRoots] = true
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
	if ref := b.sharedFieldReferences[blockRoots]; ref.Refs() > 1 {
		// Copy elements in underlying array by reference.
		roots := *b.blockRoots
		rootsCopy := roots
		r = &rootsCopy
		ref.MinusRef()
		b.sharedFieldReferences[blockRoots] = stateutil.NewRef(1)
	}

	r[idx] = blockRoot
	b.blockRoots = r

	b.markFieldAsDirty(blockRoots)
	b.addDirtyIndices(blockRoots, []uint64{idx})
	return nil
}
