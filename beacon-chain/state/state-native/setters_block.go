package state_native

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
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

	b.blockRoots.Detach(b)
	b.blockRoots = NewMultiValueBlockRoots(val)
	b.markFieldAsDirty(types.BlockRoots)
	b.rebuildTrie[types.BlockRoots] = true
	return nil
}

// UpdateBlockRootAtIndex for the beacon state. Updates the block root
// at a specific index to a new value.
func (b *BeaconState) UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if err := b.blockRoots.UpdateAt(b, idx, blockRoot); err != nil {
		return errors.Wrap(err, "could not update block roots")
	}
	b.markFieldAsDirty(types.BlockRoots)
	b.addDirtyIndices(types.BlockRoots, []uint64{idx})
	return nil
}
