package protoarray

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"go.opencensus.io/trace"
)

// This defines the minimal number of block nodes that can be in the tree
// before getting pruned upon new finalization.
const defaultPruneThreshold = 256

// This tracks the last reported head root. Used for metrics.
var lastHeadRoot [32]byte

// New initializes a new fork choice store.
func New(justifiedEpoch, finalizedEpoch types.Epoch, finalizedRoot [32]byte) *ForkChoice {
	s := &Store{
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		finalizedRoot:  finalizedRoot,
		nodes:          make([]*Node, 0),
		nodesIndices:   make(map[[32]byte]uint64),
		canonicalNodes: make(map[[32]byte]bool),
		pruneThreshold: defaultPruneThreshold,
	}

	b := make([]uint64, 0)
	v := make([]Vote, 0)

	return &ForkChoice{store: s, balances: b, votes: v}
}

// Head returns the head root from fork choice store.
// It firsts computes validator's balance changes then recalculates block tree from leaves to root.
func (f *ForkChoice) Head(
	ctx context.Context,
	justifiedEpoch types.Epoch,
	justifiedRoot [32]byte,
	justifiedStateBalances []uint64,
	finalizedEpoch types.Epoch,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.Head")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	calledHeadCount.Inc()

	newBalances := justifiedStateBalances

	// Using the write lock here because `updateCanonicalNodes` that gets called subsequently requires a write operation.
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	deltas, newVotes, err := computeDeltas(ctx, f.store.nodesIndices, f.votes, f.balances, newBalances)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not compute deltas")
	}
	f.votes = newVotes

	if err := f.store.applyWeightChanges(ctx, justifiedEpoch, finalizedEpoch, deltas); err != nil {
		return [32]byte{}, errors.Wrap(err, "Could not apply score changes")
	}
	f.balances = newBalances

	return f.store.head(ctx, justifiedRoot)
}

// ProcessAttestation processes attestation for vote accounting, it iterates around validator indices
// and update their votes accordingly.
func (f *ForkChoice) ProcessAttestation(ctx context.Context, validatorIndices []uint64, blockRoot [32]byte, targetEpoch types.Epoch) {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessAttestation")
	defer span.End()
	f.votesLock.Lock()
	defer f.votesLock.Unlock()

	for _, index := range validatorIndices {
		// Validator indices will grow the vote cache.
		for index >= uint64(len(f.votes)) {
			f.votes = append(f.votes, Vote{currentRoot: params.BeaconConfig().ZeroHash, nextRoot: params.BeaconConfig().ZeroHash})
		}

		// Newly allocated vote if the root fields are untouched.
		newVote := f.votes[index].nextRoot == params.BeaconConfig().ZeroHash &&
			f.votes[index].currentRoot == params.BeaconConfig().ZeroHash

		// Vote gets updated if it's newly allocated or high target epoch.
		if newVote || targetEpoch > f.votes[index].nextEpoch {
			f.votes[index].nextEpoch = targetEpoch
			f.votes[index].nextRoot = blockRoot
		}
	}

	processedAttestationCount.Inc()
}

// ProcessBlock processes a new block by inserting it to the fork choice store.
func (f *ForkChoice) ProcessBlock(
	ctx context.Context,
	slot types.Slot,
	blockRoot, parentRoot, graffiti [32]byte,
	justifiedEpoch, finalizedEpoch types.Epoch,
) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.ProcessBlock")
	defer span.End()

	return f.store.insert(ctx, slot, blockRoot, parentRoot, graffiti, justifiedEpoch, finalizedEpoch)
}

// Prune prunes the fork choice store with the new finalized root. The store is only pruned if the input
// root is different than the current store finalized root, and the number of the store has met prune threshold.
func (f *ForkChoice) Prune(ctx context.Context, finalizedRoot [32]byte) error {
	return f.store.prune(ctx, finalizedRoot)
}

// Nodes returns the copied list of block nodes in the fork choice store.
func (f *ForkChoice) Nodes() []*Node {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	cpy := make([]*Node, len(f.store.nodes))
	copy(cpy, f.store.nodes)
	return cpy
}

// Store returns the fork choice store object which contains all the information regarding proto array fork choice.
func (f *ForkChoice) Store() *Store {
	f.store.nodesLock.Lock()
	defer f.store.nodesLock.Unlock()
	return f.store
}

// Node returns the copied node in the fork choice store.
func (f *ForkChoice) Node(root [32]byte) *Node {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	index, ok := f.store.nodesIndices[root]
	if !ok {
		return nil
	}

	return copyNode(f.store.nodes[index])
}

// HasNode returns true if the node exists in fork choice store,
// false else wise.
func (f *ForkChoice) HasNode(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	_, ok := f.store.nodesIndices[root]
	return ok
}

// HasParent returns true if the node parent exists in fork choice store,
// false else wise.
func (f *ForkChoice) HasParent(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	i, ok := f.store.nodesIndices[root]
	if !ok || i >= uint64(len(f.store.nodes)) {
		return false
	}

	return f.store.nodes[i].parent != NonExistentNode
}

// IsCanonical returns true if the given root is part of the canonical chain.
func (f *ForkChoice) IsCanonical(root [32]byte) bool {
	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	return f.store.canonicalNodes[root]
}

// AncestorRoot returns the ancestor root of input block root at a given slot.
func (f *ForkChoice) AncestorRoot(ctx context.Context, root [32]byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArray.AncestorRoot")
	defer span.End()

	f.store.nodesLock.RLock()
	defer f.store.nodesLock.RUnlock()

	i, ok := f.store.nodesIndices[root]
	if !ok {
		return nil, errors.New("node does not exist")
	}
	if i >= uint64(len(f.store.nodes)) {
		return nil, errors.New("node index out of range")
	}

	for f.store.nodes[i].slot > slot {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		i = f.store.nodes[i].parent

		if i >= uint64(len(f.store.nodes)) {
			return nil, errors.New("node index out of range")
		}
	}

	return f.store.nodes[i].root[:], nil
}

// PruneThreshold of fork choice store.
func (s *Store) PruneThreshold() uint64 {
	return s.pruneThreshold
}

// JustifiedEpoch of fork choice store.
func (s *Store) JustifiedEpoch() types.Epoch {
	return s.justifiedEpoch
}

// FinalizedEpoch of fork choice store.
func (s *Store) FinalizedEpoch() types.Epoch {
	return s.finalizedEpoch
}

// Nodes of fork choice store.
func (s *Store) Nodes() []*Node {
	s.nodesLock.RLock()
	defer s.nodesLock.RUnlock()
	return s.nodes
}

// NodesIndices of fork choice store.
func (s *Store) NodesIndices() map[[32]byte]uint64 {
	s.nodesLock.RLock()
	defer s.nodesLock.RUnlock()
	return s.nodesIndices
}

// head starts from justified root and then follows the best descendant links
// to find the best block for head.
func (s *Store) head(ctx context.Context, justifiedRoot [32]byte) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.head")
	defer span.End()

	// Justified index has to be valid in node indices map, and can not be out of bound.
	justifiedIndex, ok := s.nodesIndices[justifiedRoot]
	if !ok {
		return [32]byte{}, errUnknownJustifiedRoot
	}
	if justifiedIndex >= uint64(len(s.nodes)) {
		return [32]byte{}, errInvalidJustifiedIndex
	}

	justifiedNode := s.nodes[justifiedIndex]
	bestDescendantIndex := justifiedNode.bestDescendant
	// If the justified node doesn't have a best descendent,
	// the best node is itself.
	if bestDescendantIndex == NonExistentNode {
		bestDescendantIndex = justifiedIndex
	}
	if bestDescendantIndex >= uint64(len(s.nodes)) {
		return [32]byte{}, errInvalidBestDescendantIndex
	}

	bestNode := s.nodes[bestDescendantIndex]

	if !s.viableForHead(bestNode) {
		return [32]byte{}, fmt.Errorf("head at slot %d with weight %d is not eligible, finalizedEpoch %d != %d, justifiedEpoch %d != %d",
			bestNode.slot, bestNode.weight/10e9, bestNode.finalizedEpoch, s.finalizedEpoch, bestNode.justifiedEpoch, s.justifiedEpoch)
	}

	// Update metrics.
	if bestNode.root != lastHeadRoot {
		headChangesCount.Inc()
		headSlotNumber.Set(float64(bestNode.slot))
		lastHeadRoot = bestNode.root
	}

	// Update canonical mapping given the head root.
	if err := s.updateCanonicalNodes(ctx, bestNode.root); err != nil {
		return [32]byte{}, err
	}

	return bestNode.root, nil
}

// updateCanonicalNodes updates the canonical nodes mapping given the input block root.
func (s *Store) updateCanonicalNodes(ctx context.Context, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "protoArrayForkChoice.updateCanonicalNodes")
	defer span.End()

	// Set the input node to canonical.
	s.canonicalNodes[root] = true

	// Get the input's parent node index.
	i := s.nodesIndices[root]
	n := s.nodes[i]
	p := n.parent

	for p != NonExistentNode {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Get the parent node, if the node is already in canonical mapping,
		// we can be sure rest of the ancestors are canonical. Exit early.
		n = s.nodes[p]
		if s.canonicalNodes[n.root] {
			break
		}

		// Set parent node to canonical. Repeat until parent node index is undefined.
		s.canonicalNodes[n.root] = true
		p = n.parent
	}

	return nil
}

// insert registers a new block node to the fork choice store's node list.
// It then updates the new node's parent with best child and descendant node.
func (s *Store) insert(ctx context.Context,
	slot types.Slot,
	root, parent, graffiti [32]byte,
	justifiedEpoch, finalizedEpoch types.Epoch) error {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.insert")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	// Return if the block has been inserted into Store before.
	if _, ok := s.nodesIndices[root]; ok {
		return nil
	}

	index := uint64(len(s.nodes))
	parentIndex, ok := s.nodesIndices[parent]
	// Mark genesis block's parent as non existent.
	if !ok {
		parentIndex = NonExistentNode
	}

	n := &Node{
		slot:           slot,
		root:           root,
		graffiti:       graffiti,
		parent:         parentIndex,
		justifiedEpoch: justifiedEpoch,
		finalizedEpoch: finalizedEpoch,
		bestChild:      NonExistentNode,
		bestDescendant: NonExistentNode,
		weight:         0,
	}

	s.nodesIndices[root] = index
	s.nodes = append(s.nodes, n)

	// Update parent with the best child and descendent only if it's available.
	if n.parent != NonExistentNode {
		if err := s.updateBestChildAndDescendant(parentIndex, index); err != nil {
			return err
		}
	}

	// Update metrics.
	processedBlockCount.Inc()
	nodeCount.Set(float64(len(s.nodes)))

	return nil
}

// applyWeightChanges iterates backwards through the nodes in store. It checks all nodes parent
// and its best child. For each node, it updates the weight with input delta and
// back propagate the nodes delta to its parents delta. After scoring changes,
// the best child is then updated along with best descendant.
func (s *Store) applyWeightChanges(ctx context.Context, justifiedEpoch, finalizedEpoch types.Epoch, delta []int) error {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.applyWeightChanges")
	defer span.End()

	// The length of the nodes can not be different than length of the delta.
	if len(s.nodes) != len(delta) {
		return errInvalidDeltaLength
	}

	// Update the justified / finalized epochs in store if necessary.
	if s.justifiedEpoch != justifiedEpoch || s.finalizedEpoch != finalizedEpoch {
		s.justifiedEpoch = justifiedEpoch
		s.finalizedEpoch = finalizedEpoch
	}

	// Iterate backwards through all index to node in store.
	for i := len(s.nodes) - 1; i >= 0; i-- {
		n := s.nodes[i]

		// There is no need to adjust the balances or manage parent of the zero hash, it
		// is an alias to the genesis block.
		if n.root == params.BeaconConfig().ZeroHash {
			continue
		}

		nodeDelta := delta[i]

		if nodeDelta < 0 {
			// A node's weight can not be negative but the delta can be negative.
			if int(n.weight)+nodeDelta < 0 {
				n.weight = 0
			} else {
				// Absolute value of node delta.
				d := nodeDelta
				if nodeDelta < 0 {
					d *= -1
				}
				// Subtract node's weight.
				n.weight -= uint64(d)
			}
		} else {
			// Add node's weight.
			n.weight += uint64(nodeDelta)
		}

		s.nodes[i] = n

		// Update parent's best child and descendent if the node has a known parent.
		if n.parent != NonExistentNode {
			// Protection against node parent index out of bound. This should not happen.
			if int(n.parent) >= len(delta) {
				return errInvalidParentDelta
			}
			// Back propagate the nodes delta to its parent.
			delta[n.parent] += nodeDelta
		}
	}

	for i := len(s.nodes) - 1; i >= 0; i-- {
		n := s.nodes[i]
		if n.parent != NonExistentNode {
			if int(n.parent) >= len(delta) {
				return errInvalidParentDelta
			}
			if err := s.updateBestChildAndDescendant(n.parent, uint64(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateBestChildAndDescendant updates parent node's best child and descendent.
// It looks at input parent node and input child node and potentially modifies parent's best
// child and best descendent indices.
// There are four outcomes:
// 1.)  The child is already the best child but it's now invalid due to a FFG change and should be removed.
// 2.)  The child is already the best child and the parent is updated with the new best descendant.
// 3.)  The child is not the best child but becomes the best child.
// 4.)  The child is not the best child and does not become best child.
func (s *Store) updateBestChildAndDescendant(parentIndex, childIndex uint64) error {

	// Protection against parent index out of bound, this should not happen.
	if parentIndex >= uint64(len(s.nodes)) {
		return errInvalidNodeIndex
	}
	parent := s.nodes[parentIndex]

	// Protection against child index out of bound, again this should not happen.
	if childIndex >= uint64(len(s.nodes)) {
		return errInvalidNodeIndex
	}
	child := s.nodes[childIndex]

	// Is the child viable to become head? Based on justification and finalization rules.
	childLeadsToViableHead, err := s.leadsToViableHead(child)
	if err != nil {
		return err
	}

	// Define 3 variables for the 3 outcomes mentioned above. This is to
	// set `parent.bestChild` and `parent.bestDescendant` to. These
	// aliases are to assist readability.
	changeToNone := []uint64{NonExistentNode, NonExistentNode}
	bestDescendant := child.bestDescendant
	if bestDescendant == NonExistentNode {
		bestDescendant = childIndex
	}
	changeToChild := []uint64{childIndex, bestDescendant}
	noChange := []uint64{parent.bestChild, parent.bestDescendant}
	var newParentChild []uint64

	if parent.bestChild != NonExistentNode {
		if parent.bestChild == childIndex && !childLeadsToViableHead {
			// If the child is already the best child of the parent but it's not viable for head,
			// we should remove it. (Outcome 1)
			newParentChild = changeToNone
		} else if parent.bestChild == childIndex {
			// If the child is already the best child of the parent, set it again to ensure best
			// descendent of the parent is updated. (Outcome 2)
			newParentChild = changeToChild
		} else {
			// Protection against parent's best child going out of bound.
			if parent.bestChild > uint64(len(s.nodes)) {
				return errInvalidBestDescendantIndex
			}
			bestChild := s.nodes[parent.bestChild]
			// Is current parent's best child viable to be head? Based on justification and finalization rules.
			bestChildLeadsToViableHead, err := s.leadsToViableHead(bestChild)
			if err != nil {
				return err
			}

			if childLeadsToViableHead && !bestChildLeadsToViableHead {
				// The child leads to a viable head, but the current parent's best child doesnt.
				newParentChild = changeToChild
			} else if !childLeadsToViableHead && bestChildLeadsToViableHead {
				// The child doesn't lead to a viable head, the current parent's best child does.
				newParentChild = noChange
			} else if child.weight == bestChild.weight {
				// If both are viable, compare their weights.
				// Tie-breaker of equal weights by root.
				if bytes.Compare(child.root[:], bestChild.root[:]) > 0 {
					newParentChild = changeToChild
				} else {
					newParentChild = noChange
				}
			} else {
				// Choose winner by weight.
				if child.weight > bestChild.weight {
					newParentChild = changeToChild
				} else {
					newParentChild = noChange
				}
			}
		}
	} else {
		if childLeadsToViableHead {
			// If parent doesn't have a best child and the child is viable.
			newParentChild = changeToChild
		} else {
			// If parent doesn't have a best child and the child is not viable.
			newParentChild = noChange
		}
	}

	// Update parent with the outcome.
	parent.bestChild = newParentChild[0]
	parent.bestDescendant = newParentChild[1]
	s.nodes[parentIndex] = parent

	return nil
}

// prune prunes the store with the new finalized root. The tree is only
// pruned if the input finalized root are different than the one in stored and
// the number of the nodes in store has met prune threshold.
func (s *Store) prune(ctx context.Context, finalizedRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "protoArrayForkChoice.prune")
	defer span.End()

	s.nodesLock.Lock()
	defer s.nodesLock.Unlock()

	// The node would have seen finalized root or else it'd
	// be able to prune it.
	finalizedIndex, ok := s.nodesIndices[finalizedRoot]
	if !ok {
		return errUnknownFinalizedRoot
	}

	// The number of the nodes has not met the prune threshold.
	// Pruning at small numbers incurs more cost than benefit.
	if finalizedIndex < s.pruneThreshold {
		return nil
	}

	// Remove the key/values from indices mapping on to be pruned nodes.
	// These nodes are before the finalized index.
	for i := uint64(0); i < finalizedIndex; i++ {
		if int(i) >= len(s.nodes) {
			return errInvalidNodeIndex
		}
		delete(s.nodesIndices, s.nodes[i].root)
	}

	// Finalized index can not be greater than the length of the node.
	if int(finalizedIndex) >= len(s.nodes) {
		return errors.New("invalid finalized index")
	}
	s.nodes = s.nodes[finalizedIndex:]

	// Adjust indices to node mapping.
	for k, v := range s.nodesIndices {
		s.nodesIndices[k] = v - finalizedIndex
	}

	// Iterate through existing nodes and adjust its parent/child indices with the newly pruned layout.
	for i, node := range s.nodes {
		if node.parent != NonExistentNode {
			// If the node's parent is less than finalized index, set it to non existent.
			if node.parent >= finalizedIndex {
				node.parent -= finalizedIndex
			} else {
				node.parent = NonExistentNode
			}
		}
		if node.bestChild != NonExistentNode {
			if node.bestChild < finalizedIndex {
				return errInvalidBestChildIndex
			}
			node.bestChild -= finalizedIndex
		}
		if node.bestDescendant != NonExistentNode {
			if node.bestDescendant < finalizedIndex {
				return errInvalidBestDescendantIndex
			}
			node.bestDescendant -= finalizedIndex
		}

		s.nodes[i] = node
	}

	prunedCount.Inc()

	return nil
}

// leadsToViableHead returns true if the node or the best descendent of the node is viable for head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) leadsToViableHead(node *Node) (bool, error) {
	var bestDescendentViable bool
	bestDescendentIndex := node.bestDescendant

	// If the best descendant is not part of the leaves.
	if bestDescendentIndex != NonExistentNode {
		// Protection against out of bound, best descendent index can not be
		// exceeds length of nodes list.
		if bestDescendentIndex >= uint64(len(s.nodes)) {
			return false, errInvalidBestDescendantIndex
		}

		bestDescendentNode := s.nodes[bestDescendentIndex]
		bestDescendentViable = s.viableForHead(bestDescendentNode)
	}

	// The node is viable as long as the best descendent is viable.
	return bestDescendentViable || s.viableForHead(node), nil
}

// viableForHead returns true if the node is viable to head.
// Any node with diff finalized or justified epoch than the ones in fork choice store
// should not be viable to head.
func (s *Store) viableForHead(node *Node) bool {
	// `node` is viable if its justified epoch and finalized epoch are the same as the one in `Store`.
	// It's also viable if we are in genesis epoch.
	justified := s.justifiedEpoch == node.justifiedEpoch || s.justifiedEpoch == 0
	finalized := s.finalizedEpoch == node.finalizedEpoch || s.finalizedEpoch == 0

	return justified && finalized
}
