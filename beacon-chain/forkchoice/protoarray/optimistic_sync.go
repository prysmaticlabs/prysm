package protoarray

import (
	"context"
)

// pruneInvalid removes the node with the given root and all of its children
// from the Fork Choice Store.
func (s *Store) pruneInvalid(ctx context.Context, root [32]byte) error {
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
				if i == len(children)-1 {
					node.parent.children = children[:len(children)-2]
				} else {
					children[i] = children[len(children)-1]
					node.parent.children = children[:len(children)-2]
				}
				break
			}
		}
	}
	return s.removeSubtree(ctx, node)
}

// removeSubtree removes `node` and all of its descendant from the Store
func (s *Store) removeSubtree(ctx context.Context, node *Node) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	for _, child := range node.children {
		if err := s.removeSubtree(ctx, child); err != nil {
			return err
		}
	}

	delete(s.nodeByRoot, node.root)
	delete(s.canonicalNodes, node.root)
	return nil
}
