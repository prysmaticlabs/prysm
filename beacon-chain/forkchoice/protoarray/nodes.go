package protoarray

import (
	"context"

	"go.opencensus.io/trace"
)

// leadsToViableHead returns true if the node or the best descendent of the node is viable for head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) leadsToViableHead(ctx context.Context, node *Node) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.leadsToViableHead")
	defer span.End()

	var bestDescendentViable bool
	bestDescendentIndex := node.bestDescendant

	// If the best descendant is not part of the leaves.
	if bestDescendentIndex != nonExistentNode {
		// Protection against out of bound, best descendent index can not be
		// exceeds length of nodes list.
		if bestDescendentIndex >= uint64(len(s.nodes)) {
			return false, errInvalidBestDescendantIndex
		}

		bestDescendentNode := s.nodes[bestDescendentIndex]
		bestDescendentViable = s.viableForHead(ctx, bestDescendentNode)
	}

	// The node is viable as long as the best descendent is viable.
	return bestDescendentViable || s.viableForHead(ctx, node), nil
}

// viableForHead returns true if the node is viable to head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) viableForHead(ctx context.Context, node *Node) bool {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.viableForHead")
	defer span.End()

	// `node` is viable if its justified epoch and finalized epoch are the same as the one in `Store`.
	// It's also viable if we are in genesis epoch.
	justified := s.justifiedEpoch == node.justifiedEpoch || s.justifiedEpoch == 0
	finalized := s.finalizedEpoch == node.finalizedEpoch || s.finalizedEpoch == 0

	return justified && finalized
}
