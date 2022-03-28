package protoarray

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
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

// IsOptimistic returns true if this node is optimistically synced
// A optimistically synced block is synced as usual, but its
// execution payload is not validated, while the EL is still syncing.
// This function returns an error if the block is not found in the fork choice
// store
func (f *ForkChoice) IsOptimistic(ctx context.Context, root [32]byte) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	f.store.nodesLock.RLock()
	index, ok := f.store.nodesIndices[root]
	if !ok {
		f.store.nodesLock.RUnlock()
		return false, ErrUnknownNodeRoot
	}
	node := f.store.nodes[index]
	slot := node.slot

	// If the node is a synced tip, then it's fully validated
	f.syncedTips.RLock()
	_, ok = f.syncedTips.validatedTips[root]
	if ok {
		f.syncedTips.RUnlock()
		f.store.nodesLock.RUnlock()
		return false, nil
	}
	f.syncedTips.RUnlock()

	// If the slot is higher than the max synced tip, it's optimistic
	min, max := f.boundarySyncedTips()
	if slot > max {
		f.store.nodesLock.RUnlock()
		return true, nil
	}

	// If the slot is lower than the min synced tip, it's fully validated
	if slot <= min {
		f.store.nodesLock.RUnlock()
		return false, nil
	}

	// if the node is a leaf of the Fork Choice tree, then it's
	// optimistic
	childIndex := node.BestChild()
	if childIndex == NonExistentNode {
		f.store.nodesLock.RUnlock()
		return true, nil
	}

	// recurse to the child
	child := f.store.nodes[childIndex]
	root = child.root
	f.store.nodesLock.RUnlock()
	return f.IsOptimistic(ctx, root)
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

// SetOptimisticToValid is called with the root of a block that was returned as
// VALID by the EL. This routine recomputes and updates the synced_tips map to
// account for this new tip.
// WARNING: This method returns an error if the root is not found in forkchoice or
// if the root is not a leaf of the fork choice tree.
func (f *ForkChoice) SetOptimisticToValid(ctx context.Context, root [32]byte) error {
	f.store.nodesLock.RLock()
	// We can only update if given root is in Fork Choice
	index, ok := f.store.nodesIndices[root]
	if !ok {
		return errInvalidNodeIndex
	}
	node := f.store.nodes[index]
	f.store.nodesLock.RUnlock()

	// Stop early if the node is Valid
	optimistic, err := f.IsOptimistic(ctx, root)
	if err != nil {
		return err
	}
	if !optimistic {
		return nil
	}
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	// Cache root and slot to validated tips
	newTips := make(map[[32]byte]types.Slot)
	newValidSlot := node.slot
	newTips[root] = newValidSlot

	// Compute the full valid path from the given node to its previous synced tip
	// This path will now consist of fully validated blocks. Notice that
	// the previous tip may have been outside the Fork Choice store.
	// In this case, only one block can be in syncedTips as the whole
	// Fork Choice would be a descendant of this block.
	validPath := make(map[uint64]bool)
	validPath[index] = true
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
	leaves, err := f.store.leaves()
	if err != nil {
		return err
	}

	// For each leaf, recompute the new tip.
	for _, i := range leaves {
		node = f.store.nodes[i]
		j := i
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}
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
	lastSyncedTipSlot.Set(float64(newValidSlot))
	syncedTipsCount.Set(float64(len(newTips)))
	return nil
}

//  SetOptimisticToInvalid updates the synced_tips map when the block with the given root becomes INVALID.
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root [32]byte) ([][32]byte, error) {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	invalidRoots := make([][32]byte, 0)
	idx, ok := f.store.nodesIndices[root]
	if !ok {
		return invalidRoots, errInvalidNodeIndex
	}
	node := f.store.nodes[idx]
	// We only support changing status for the tips in Fork Choice store.
	if node.bestChild != NonExistentNode {
		return invalidRoots, errInvalidNodeIndex
	}

	parentIndex := node.parent
	// This should not happen
	if parentIndex == NonExistentNode {
		return invalidRoots, errInvalidNodeIndex
	}
	// Update the weights of the nodes subtracting the INVALID node's weight
	weight := node.weight
	node = f.store.nodes[parentIndex]
	for {
		if ctx.Err() != nil {
			return invalidRoots, ctx.Err()
		}
		node.weight -= weight
		if node.parent == NonExistentNode {
			break
		}
		node = f.store.nodes[node.parent]
	}
	parent := copyNode(f.store.nodes[parentIndex])

	// delete the invalid node, order is important
	f.store.nodes = append(f.store.nodes[:idx], f.store.nodes[idx+1:]...)
	delete(f.store.nodesIndices, root)
	invalidRoots = append(invalidRoots, root)
	// Fix parent and best child for each node
	for _, node := range f.store.nodes {
		if node.parent == NonExistentNode {
			node.parent = NonExistentNode
		} else if node.parent > idx {
			node.parent -= 1
		}
		if node.bestChild == NonExistentNode || node.bestChild == idx {
			node.bestChild = NonExistentNode
		} else if node.bestChild > idx {
			node.bestChild -= 1
		}
		if node.bestDescendant == NonExistentNode || node.bestDescendant == idx {
			node.bestDescendant = NonExistentNode
		} else if node.bestDescendant > idx {
			node.bestDescendant -= 1
		}
	}

	// Update the parent's best child and best descendant if necessary.
	if parent.bestChild == idx || parent.bestDescendant == idx {
		for childIndex, child := range f.store.nodes {
			if child.parent == parentIndex {
				err := f.store.updateBestChildAndDescendant(
					parentIndex, uint64(childIndex))
				if err != nil {
					return invalidRoots, err
				}
				break
			}
		}
	}

	// Return early if the parent is not a synced_tip.
	f.syncedTips.Lock()
	defer f.syncedTips.Unlock()
	parentRoot := parent.root
	_, ok = f.syncedTips.validatedTips[parentRoot]
	if !ok {
		return invalidRoots, nil
	}

	leaves, err := f.store.leaves()
	if err != nil {
		return invalidRoots, err
	}

	for _, i := range leaves {
		node = f.store.nodes[i]
		for {
			if ctx.Err() != nil {
				return invalidRoots, ctx.Err()
			}

			// Return early if the parent is still a synced tip
			if node.root == parentRoot {
				return invalidRoots, nil
			}
			_, ok = f.syncedTips.validatedTips[node.root]
			if ok {
				break
			}
			if node.parent == NonExistentNode {
				break
			}
			node = f.store.nodes[node.parent]
		}
	}
	delete(f.syncedTips.validatedTips, parentRoot)
	syncedTipsCount.Set(float64(len(f.syncedTips.validatedTips)))
	return invalidRoots, nil
}
