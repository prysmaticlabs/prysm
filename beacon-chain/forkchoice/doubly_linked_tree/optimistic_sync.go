package doubly_linked_tree

import (
	"context"
)

// removeNode removes the node with the given root and all of its children
// from the Fork Choice Store.
func (s *Store) removeNode(ctx context.Context, root [32]byte) error {
	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	node, ok := s.nodeByRoot[root]
	if !ok || node == nil {
		return errNilNode
	}
	if !node.optimistic || node.parent == nil {
		return errInvalidOptimisticStatus
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
	return s.removeNodeAndChildren(ctx, node)
}

// removeNodeAndChildren removes `node` and all of its descendant from the Store
func (s *Store) removeNodeAndChildren(ctx context.Context, node *Node) error {
	for _, child := range node.children {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.removeNodeAndChildren(ctx, child); err != nil {
			return err
		}
	}

	delete(s.nodeByRoot, node.root)
	return nil
}
