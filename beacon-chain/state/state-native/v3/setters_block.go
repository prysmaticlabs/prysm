package v3

import (
	"fmt"

	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
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
func (b *BeaconState) SetBlockRoots(val [][]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	var rootsArr [fieldparams.BlockRootsLength][fieldparams.RootLength]byte
	for i := 0; i < len(rootsArr); i++ {
		copy(rootsArr[i][:], val[i])
	}
	roots := customtypes.BlockRoots{}
	roots.SetFromBaseField(rootsArr)
	b.blockRoots = &roots
	b.markFieldAsDirty(blockRoots)
	b.rebuildTrie[blockRoots] = true
	return nil
}

// UpdateBlockRootAtIndex for the beacon state. Updates the block root
// at a specific index to a new value.
func (b *BeaconState) UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error {
	if b.blockRoots.TotalLength() <= idx {
		return fmt.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.blockRoots.SetRootAtIndex(idx, blockRoot)

	b.markFieldAsDirty(blockRoots)
	b.addDirtyIndices(blockRoots, []uint64{idx})
	return nil
}
