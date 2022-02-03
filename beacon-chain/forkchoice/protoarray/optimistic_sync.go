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

// This function returns the index of sync tip node that's ancestor to the input node.
// In the event of none, `NonExistentNode` is returned.
// This internal method assumes the caller holds a lock on syncedTips and s.nodesLock
func (s *Store) findSyncedTip(ctx context.Context, node *Node, syncedTips *optimisticStore) (uint64, error) {
	for {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if _, ok := syncedTips.validatedTips[node.root]; ok {
			return s.nodesIndices[node.root], nil
		}
		if node.parent == NonExistentNode {
			return NonExistentNode, nil
		}
		node = s.nodes[node.parent]
	}
}

// UpdateSyncedTips updates the synced_tips map when the block with the given root becomes VALID.
func (f *ForkChoice) UpdateSyncedTips(ctx context.Context, root [32]byte) error {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	// We can only update if given root is in fork choice
	index, ok := f.store.nodesIndices[root]
	if !ok {
		return errInvalidNodeIndex
	}

	// We can only update if root is a leaf in fork choice
	node := f.store.nodes[index]
	if node.bestChild != NonExistentNode {
		return errInvalidBestChildIndex
	}

	// Stop early if the root is part of validated tips
	f.syncedTips.Lock()
	defer f.syncedTips.Unlock()
	_, ok = f.syncedTips.validatedTips[root]
	if ok {
		return nil
	}

	// Cache root and slot to validated tips
	f.syncedTips.validatedTips[root] = node.slot

	// Compute the full valid path from the given node to its previous synced tip
	// This path will now consist of fully validated blocks. Notice that
	// the previous tip may have been outside the Fork Choice store.
	// In this case, only one block can be in syncedTips as the whole
	// Fork Choice would be a descendant of this block.
	validPath := make(map[uint64]bool)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		parentIndex := node.parent
		if parentIndex == NonExistentNode {
			break
		}
		if parentIndex >= uint64(len(f.store.nodes)) {
			return errInvalidNodeIndex
		}
		node = f.store.nodes[parentIndex]
		_, ok = f.syncedTips.validatedTips[node.root]
		if ok {
			break
		}
		validPath[parentIndex] = true
	}

	// Retrieve the list of leaves in the Fork Choice
	// These are all the nodes that have NonExistentNode as best child.
	var leaves []uint64
	for i := uint64(0); i < uint64(len(f.store.nodes)); i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		node = f.store.nodes[i]
		if node.bestChild == NonExistentNode {
			leaves = append(leaves, i)
		}
	}

	// For each leaf, recompute the new tip.
	newTips := make(map[[32]byte]types.Slot)
	for _, i := range leaves {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		node = f.store.nodes[i]
		j := i
		for {
			// Stop if we reached the previous tip
			_, ok = f.syncedTips.validatedTips[node.root]
			if ok {
				newTips[node.root] = node.slot
				break
			}

			// Stop if we reach valid path
			_, ok = validPath[j]
			if ok {
				newTips[node.root] = node.slot
				break
			}

			j = node.parent
			if j == NonExistentNode {
				break
			}
			if j >= uint64(len(f.store.nodes)) {
				return errInvalidNodeIndex
			}
			node = f.store.nodes[j]
		}
	}

	f.syncedTips.validatedTips = newTips
	return nil
}
