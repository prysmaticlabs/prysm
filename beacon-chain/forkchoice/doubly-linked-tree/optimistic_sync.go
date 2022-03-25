package doublylinkedtree

import (
	"context"
)

// removeNode removes the node with the given root and all of its children
// from the Fork Choice Store.
func (s *Store) removeNode(ctx context.Context, root [32]byte) ([][32]byte, error) {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()
	invalidRoots := make([][32]byte, 0)

	node, ok := s.nodeByRoot[root]
	if !ok || node == nil {
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
				node.parent.children = children[:len(children)-2]
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
	delete(s.nodeByRoot, node.root)
	return invalidRoots, nil
}
