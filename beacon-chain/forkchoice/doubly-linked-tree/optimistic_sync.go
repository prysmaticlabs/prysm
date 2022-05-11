package doublylinkedtree

import (
	"context"

	"github.com/prysmaticlabs/prysm/config/params"
)

func (s *Store) setOptimisticToInvalid(ctx context.Context, root, parentRoot, payloadHash [32]byte) ([][32]byte, error) {
	s.nodesLock.Lock()
	invalidRoots := make([][32]byte, 0)
	node, ok := s.nodeByRoot[root]
	if !ok {
		node, ok = s.nodeByRoot[parentRoot]
		if !ok || node == nil {
			s.nodesLock.Unlock()
			return invalidRoots, ErrNilNode
		}
		// return early if the parent is LVH
		if node.payloadHash == payloadHash {
			s.nodesLock.Unlock()
			return invalidRoots, nil
		}
	} else {
		if node == nil {
			s.nodesLock.Unlock()
			return invalidRoots, ErrNilNode
		}
		if node.parent.root != parentRoot {
			s.nodesLock.Unlock()
			return invalidRoots, errInvalidParentRoot
		}
	}
	// Check if last valid hash is an ancestor of the passed node.
	lastValid, ok := s.nodeByPayload[payloadHash]
	if !ok || lastValid == nil {
		s.nodesLock.Unlock()
		return invalidRoots, errUnknownPayloadHash
	}
	firstInvalid := node
	for ; firstInvalid.parent != nil && firstInvalid.parent.payloadHash != payloadHash; firstInvalid = firstInvalid.parent {
		if ctx.Err() != nil {
			s.nodesLock.Unlock()
			return invalidRoots, ctx.Err()
		}
	}
	// Deal with the case that the last valid payload is in a different fork
	// This means we are dealing with an EE that does not follow the spec
	if firstInvalid.parent == nil {
		// return early if the invalid node was not imported
		if node.root == parentRoot {
			s.nodesLock.Unlock()
			return invalidRoots, nil
		}
		firstInvalid = node
	}
	s.nodesLock.Unlock()
	return s.removeNode(ctx, firstInvalid)
}

// removeNode removes the node with the given root and all of its children
// from the Fork Choice Store.
func (s *Store) removeNode(ctx context.Context, node *Node) ([][32]byte, error) {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	invalidRoots := make([][32]byte, 0)

	if node == nil {
		return invalidRoots, ErrNilNode
	}
	if !node.optimistic || node.parent == nil {
		return invalidRoots, errInvalidOptimisticStatus
	}

	children := node.parent.children
	if len(children) == 1 {
		node.parent.children = []*Node{}
	} else {
		for i, n := range children {
			if n == node {
				if i != len(children)-1 {
					children[i] = children[len(children)-1]
				}
				node.parent.children = children[:len(children)-1]
				break
			}
		}
	}
	return s.removeNodeAndChildren(ctx, node, invalidRoots)
}

// removeNodeAndChildren removes `node` and all of its descendant from the Store
func (s *Store) removeNodeAndChildren(ctx context.Context, node *Node, invalidRoots [][32]byte) ([][32]byte, error) {
	var err error
	for _, child := range node.children {
		if ctx.Err() != nil {
			return invalidRoots, ctx.Err()
		}
		if invalidRoots, err = s.removeNodeAndChildren(ctx, child, invalidRoots); err != nil {
			return invalidRoots, err
		}
	}
	invalidRoots = append(invalidRoots, node.root)
	s.proposerBoostLock.Lock()
	if node.root == s.proposerBoostRoot {
		s.proposerBoostRoot = [32]byte{}
	}
	if node.root == s.previousProposerBoostRoot {
		s.previousProposerBoostRoot = params.BeaconConfig().ZeroHash
		s.previousProposerBoostScore = 0
	}
	s.proposerBoostLock.Unlock()
	delete(s.nodeByRoot, node.root)
	delete(s.nodeByPayload, node.payloadHash)
	return invalidRoots, nil
}
