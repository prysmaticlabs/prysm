package protoarray

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

// This returns the minimum and maximum slot of the synced_tips tree
func (f *ForkChoice) boundarySyncedTips() (types.Slot, types.Slot) {
	f.syncedTips.RLock()
	defer f.syncedTips.RUnlock()

	min := params.BeaconConfig().FarFutureSlot
	max := types.Slot(0)
	for _, slot := range f.syncedTips.validatedTips {
		if slot > max {
			max = slot
		}
		if slot < min {
			min = slot
		}
	}
	return min, max
}

// Optimistic returns true if this node is optimistically synced
// A optimistically synced block is synced as usual, but its
// execution payload is not validated, while the EL is still syncing.
// WARNING: this function does not check if slot corresponds to the
//          block with the given root. An incorrect response may be
//          returned when requesting earlier than finalized epoch due
//          to pruning of non-canonical branches. A requests for a
//          combination root/slot of an available block is guaranteed
//          to yield the correct result. The caller is responsible for
//          checking the block's availability. A consensus bug could be
//          a cause of getting this wrong, so think twice before passing
//          a wrong pair.
func (f *ForkChoice) Optimistic(ctx context.Context, root [32]byte, slot types.Slot) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	// If the node is a synced tip, then it's fully validated
	f.syncedTips.RLock()
	_, ok := f.syncedTips.validatedTips[root]
	if ok {
		return false, nil
	}
	f.syncedTips.RUnlock()

	// If the slot is higher than the max synced tip, it's optimistic
	min, max := f.boundarySyncedTips()
	if slot > max {
		return true, nil
	}

	// If the slot is lower than the min synced tip, it's fully validated
	if slot <= min {
		return false, nil
	}

	// If we reached this point then the block has to be in the Fork Choice
	// Store!
	f.store.nodesLock.RLock()
	index, ok := f.store.nodesIndices[root]
	if !ok {
		// This should not happen
		f.store.nodesLock.RUnlock()
		return false, fmt.Errorf("invalid root, slot combination, got %#x, %d",
			bytesutil.Trunc(root[:]), slot)
	}
	node := f.store.nodes[index]

	// if the node is a leaf of the Fork Choice tree, then it's
	// optimistic
	childIndex := node.BestChild()
	if childIndex == NonExistentNode {
		return true, nil
	}

	// recurse to the child
	child := f.store.nodes[childIndex]
	root = child.root
	slot = child.slot
	f.store.nodesLock.RUnlock()
	return f.Optimistic(ctx, root, slot)
}
