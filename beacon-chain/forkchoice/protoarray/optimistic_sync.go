package protoarray

import (
	"context"
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
		return false, ErrUnknownNodeRoot
	}
	node := f.store.nodes[index]
	return node.optimistic == SYNCING, nil
}

// SetOptimisticToValid is called with the root of a block that was returned as
// VALID by the EL.
// WARNING: This method returns an error if the root is not found in forkchoice
func (f *ForkChoice) SetOptimisticToValid(ctx context.Context, root [32]byte) error {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()
	// We can only update if given root is in Fork Choice
	index, ok := f.store.nodesIndices[root]
	if !ok {
		return ErrUnknownNodeRoot
	}

	for node := f.store.nodes[index]; node.optimistic == SYNCING; node = f.store.nodes[index] {
		node.optimistic = VALID
		index = node.parent
		if index == NonExistentNode {
			break
		}
		validatedNodesCount.Inc()
	}
	return nil
}

// SetOptimisticToInvalid updates the synced_tips map when the block with the given root becomes INVALID.
// It takes two parameters: the root of the INVALID block and the payload Hash
// of the last valid block.s
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root, payload [32]byte) ([][32]byte, error) {
	f.store.nodesLock.Lock()
	invalidRoots := make([][32]byte, 0)
	// We only support setting invalid a node existing in Forkchoice
	invalidIndex, ok := f.store.nodesIndices[root]
	if !ok {
		f.store.nodesLock.Unlock()
		return invalidRoots, ErrUnknownNodeRoot
	}
	node := f.store.nodes[invalidIndex]

	lastValidIndex, ok := f.store.payloadIndices[payload]
	if !ok || lastValidIndex == NonExistentNode {
		f.store.nodesLock.Unlock()
		return invalidRoots, errInvalidFinalizedNode
	}

	// Check if last valid hash is an ancestor of the passed node
	firstInvalidIndex := node.parent
	for ; firstInvalidIndex != NonExistentNode && firstInvalidIndex != lastValidIndex; firstInvalidIndex = node.parent {
		if ctx.Err() != nil {
			f.store.nodesLock.Unlock()
			return invalidRoots, ctx.Err()
		}
		node = f.store.nodes[firstInvalidIndex]
	}
	if node.parent != lastValidIndex {
		node = f.store.nodes[invalidIndex]
		firstInvalidIndex = invalidIndex
		lastValidIndex = node.parent
		if lastValidIndex == NonExistentNode {
			f.store.nodesLock.Unlock()
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
		if ctx.Err() != nil {
			f.store.nodesLock.Unlock()
			return invalidRoots, ctx.Err()
		}
		validNode.weight -= weight
	}

	// Remove the invalid roots from our store maps and adjust their weight
	// to zero
	invalidIndices := map[uint64]bool{firstInvalidIndex: true}
	node.optimistic = INVALID
	node.weight = 0
	delete(f.store.nodesIndices, node.root)
	delete(f.store.canonicalNodes, node.root)
	delete(f.store.payloadIndices, node.payloadHash)
	for index := firstInvalidIndex + 1; index < uint64(len(f.store.nodes)); index++ {
		invalidNode := f.store.nodes[index]
		if _, ok := invalidIndices[invalidNode.parent]; !ok {
			continue
		}
		if invalidNode.optimistic == VALID {
			f.store.nodesLock.Unlock()
			return invalidRoots, errInvalidOptimisticStatus
		}
		invalidNode.optimistic = INVALID
		invalidIndices[index] = true
		invalidNode.weight = 0
		delete(f.store.nodesIndices, invalidNode.root)
		delete(f.store.canonicalNodes, invalidNode.root)
		delete(f.store.payloadIndices, invalidNode.payloadHash)
	}

	for index := range invalidIndices {
		invalidRoots = append(invalidRoots, f.store.nodes[index].root)
	}

	// Update best child and descendant
	for i := len(f.store.nodes) - 1; i >= 0; i-- {
		n := f.store.nodes[i]
		if n.parent != NonExistentNode {
			if err := f.store.updateBestChildAndDescendant(n.parent, uint64(i)); err != nil {
				f.store.nodesLock.Unlock()
				return invalidRoots, err
			}
		}
	}

	lastValidRoot := f.store.nodes[lastValidIndex].root
	f.store.nodesLock.Unlock()
	return invalidRoots, f.SetOptimisticToValid(ctx, lastValidRoot)
}
