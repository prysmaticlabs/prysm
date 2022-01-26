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

// This updates the synced_tips map when the block with the given root becomes
// VALID
func (f *ForkChoice) UpdateSyncedTips(ctx context.Context, root [32]byte) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// We can only change status of blocks already in the Fork Choice
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	index, ok := f.store.nodesIndices[root]
	if !ok {
		return errInvalidNodeIndex
	}

	// We can only update from a head, no intermediate nodes
	// Note: this can be relaxed, but the complexity of code to deal with
	// this case is unnecessary.
	node := f.store.nodes[index]
	if node.bestChild != NonExistentNode {
		return errInvalidBestChildIndex
	}

	// Stop early if the block was already VALID
	f.syncedTips.Lock()
	defer f.syncedTips.Unlock()
	_, ok = f.syncedTips.validatedTips[root]
	if ok {
		return nil
	}

	// The new VALID node will be a synced tip
	f.syncedTips.validatedTips[root] = node.slot

	// Compute the full path from the given node to its synced tip
	// This path will now consist of fully validated blocks. Notice that
	// the previous tip may have been outside of the Fork Choice store.
	// In this case, only one block can be in syncedTips as the whole
	// Fork Choice would be a descendant of this block.
	cnode := node
	validPath := make(map[uint64]bool)
	for {
		index := cnode.parent
		if index == NonExistentNode {
			break
		}
		cnode = f.store.nodes[index]
		_, ok = f.syncedTips.validatedTips[cnode.root]
		if ok {
			break
		}
		validPath[index] = true
	}

	// Compute the list of leaves in the Fork Choice
	// These are all the nodes that have NonExistentNode as best child.
	leaves := []uint64{}
	for i := uint64(0); i < uint64(len(f.store.nodes)); i++ {
		node = f.store.nodes[i]
		if node.bestChild == NonExistentNode {
			leaves = append(leaves, i)
		}
	}

	// For each leaf recompute it's new tip.
	newTips := make(map[[32]byte]types.Slot)
	for _, i := range leaves {
		node = f.store.nodes[i]
		idx := uint64(i)
		for {
			// Stop if we reached the previous tip
			_, ok = f.syncedTips.validatedTips[node.root]
			if ok {
				newTips[node.root] = node.slot
				break
			}

			// Stop if we reach a new valid tip
			_, ok = validPath[idx]
			if ok {
				newTips[node.root] = node.slot
				break
			}

			idx = node.parent
			if idx == NonExistentNode {
				break
			}
			node = f.store.nodes[idx]
		}
	}

	// Add the new tips to syncedTips
	f.syncedTips.validatedTips = newTips
	return nil
}
