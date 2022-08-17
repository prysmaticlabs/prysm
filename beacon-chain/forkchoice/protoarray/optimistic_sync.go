package protoarray

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/config/params"
)

// IsOptimistic returns true if this node is optimistically synced
// A optimistically synced block is synced as usual, but its
// execution payload is not validated, while the EL is still syncing.
// This function returns an error if the block is not found in the fork choice
// store
func (f *ForkChoice) IsOptimistic(root [32]byte) (bool, error) {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	index, ok := f.store.nodesIndices[root]
	if !ok {
		return true, ErrUnknownNodeRoot
	}
	node := f.store.nodes[index]
	return node.status == syncing, nil
}

// SetOptimisticToValid is called with the root of a block that was returned as
// VALID by the EL.
//
// WARNING: This method returns an error if the root is not found in forkchoice
func (f *ForkChoice) SetOptimisticToValid(ctx context.Context, root [32]byte) error {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	// We can only update if given root is in Fork Choice
	index, ok := f.store.nodesIndices[root]
	if !ok {
		return ErrUnknownNodeRoot
	}

	for node := f.store.nodes[index]; node.status == syncing; node = f.store.nodes[index] {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		node.status = valid
		index = node.parent
		if index == NonExistentNode {
			break
		}
	}
	return nil
}

// SetOptimisticToInvalid updates the synced_tips map when the block with the given root becomes INVALID.
// It takes three parameters: the root of the INVALID block, its parent root and the payload Hash
// of the last valid block
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root, parentRoot, payloadHash [32]byte) ([][32]byte, error) {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	invalidRoots := make([][32]byte, 0)
	lastValidIndex, ok := f.store.payloadIndices[payloadHash]
	if !ok {
		lastValidIndex = uint64(len(f.store.nodes))
	}

	invalidIndex, ok := f.store.nodesIndices[root]
	if !ok {
		invalidIndex, ok = f.store.nodesIndices[parentRoot]
		if !ok {
			return invalidRoots, ErrUnknownNodeRoot
		}
		// return early if parent is LVH
		if invalidIndex == lastValidIndex {
			return invalidRoots, nil
		}
	}
	node := f.store.nodes[invalidIndex]

	// Check if last valid hash is an ancestor of the passed node
	firstInvalidIndex := node.parent
	for ; firstInvalidIndex != NonExistentNode && firstInvalidIndex != lastValidIndex; firstInvalidIndex = node.parent {
		node = f.store.nodes[firstInvalidIndex]
	}

	// Deal with the case that the last valid payload is in a different fork
	// This means we are dealing with an EE that does not follow the spec
	if node.parent != lastValidIndex {
		node = f.store.nodes[invalidIndex]
		// return early if invalid node was not imported
		if node.root == parentRoot {
			return invalidRoots, nil
		}

		firstInvalidIndex = invalidIndex
		lastValidIndex = node.parent
		if lastValidIndex == NonExistentNode {
			return invalidRoots, errInvalidFinalizedNode
		}
	} else {
		firstInvalidIndex = f.store.nodesIndices[node.root]
	}

	// Update the weights of the nodes subtracting the first INVALID node's weight
	weight := node.weight
	var validNode *Node
	for index := lastValidIndex; index != NonExistentNode; index = validNode.parent {
		validNode = f.store.nodes[index]
		validNode.weight -= weight
	}

	// Find the current proposer boost (it should be set to zero if an
	// INVALID block was boosted)
	f.store.proposerBoostLock.RLock()
	boostRoot := f.store.proposerBoostRoot
	previousBoostRoot := f.store.previousProposerBoostRoot
	f.store.proposerBoostLock.RUnlock()

	// Remove the invalid roots from our store maps and adjust their weight
	// to zero
	boosted := node.root == boostRoot
	previouslyBoosted := node.root == previousBoostRoot

	invalidIndices := map[uint64]bool{firstInvalidIndex: true}
	node.status = invalid
	node.weight = 0
	delete(f.store.nodesIndices, node.root)
	delete(f.store.canonicalNodes, node.root)
	delete(f.store.payloadIndices, node.payloadHash)
	for index := firstInvalidIndex + 1; index < uint64(len(f.store.nodes)); index++ {
		invalidNode := f.store.nodes[index]
		if _, ok := invalidIndices[invalidNode.parent]; !ok {
			continue
		}
		if invalidNode.status == valid {
			return invalidRoots, errInvalidOptimisticStatus
		}
		if !boosted && invalidNode.root == boostRoot {
			boosted = true
		}
		if !previouslyBoosted && invalidNode.root == previousBoostRoot {
			previouslyBoosted = true
		}
		invalidNode.status = invalid
		invalidIndices[index] = true
		invalidNode.weight = 0
		delete(f.store.nodesIndices, invalidNode.root)
		delete(f.store.canonicalNodes, invalidNode.root)
		delete(f.store.payloadIndices, invalidNode.payloadHash)
	}
	if boosted {
		if err := f.ResetBoostedProposerRoot(ctx); err != nil {
			return invalidRoots, err
		}
	}
	if previouslyBoosted {
		f.store.proposerBoostLock.Lock()
		f.store.previousProposerBoostRoot = params.BeaconConfig().ZeroHash
		f.store.previousProposerBoostScore = 0
		f.store.proposerBoostLock.Unlock()
	}

	for index := range invalidIndices {
		invalidRoots = append(invalidRoots, f.store.nodes[index].root)
	}

	// Update the best child and descendant
	for i := len(f.store.nodes) - 1; i >= 0; i-- {
		n := f.store.nodes[i]
		if n.parent != NonExistentNode {
			if err := f.store.updateBestChildAndDescendant(n.parent, uint64(i)); err != nil {
				return invalidRoots, err
			}
		}
	}
	return invalidRoots, nil
}

// AllTipsAreInvalid returns true if no forkchoice tip is viable for head
func (f *ForkChoice) AllTipsAreInvalid() bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	return f.store.allTipsAreInvalid
}
