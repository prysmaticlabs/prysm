package decoupled

import (
	"context"
	"slices"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice"
	doublylinkedtree "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/doubly-linked-tree"
	forkchoicetypes "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/types"
	"github.com/OffchainLabs/prysm/v7/config/params"
	forkchoice2 "github.com/OffchainLabs/prysm/v7/consensus-types/forkchoice"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	prysmTime "github.com/OffchainLabs/prysm/v7/time"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

var errNoDump = errors.New("the old fork choice dump is gone after the Decoupled fork")

func checkpoint(n *Node) *forkchoicetypes.Checkpoint {
	return &forkchoicetypes.Checkpoint{Epoch: slots.ToEpoch(n.Slot), Root: n.Root}
}

// checkpointHash fixes the payload a block builds on, not its own, so since Gloas it is the parent hash.
func (f *ForkChoice) checkpointHash(root [32]byte) [32]byte {
	p := f.exec[root]
	switch {
	case p == nil:
		return [32]byte{}
	case p.preGloas:
		return p.blockHash
	default:
		return p.parentHash
	}
}

// ancestor falls back to the database below F, which every chain shares.
func (f *ForkChoice) ancestor(ctx context.Context, st *Store, root [32]byte, slot primitives.Slot) ([32]byte, error) {
	if !st.HasNode(root) {
		return [32]byte{}, doublylinkedtree.ErrNilNode
	}
	if r, ok := st.AncestorAtSlot(root, slot); ok {
		return r, nil
	}
	if f.ancestorByDB == nil {
		return [32]byte{}, doublylinkedtree.ErrNilNode
	}
	return f.ancestorByDB(ctx, st.finalized.Root, slot)
}

// GetProposerHead returns the published head after the fork.
func (f *ForkChoice) GetProposerHead() [32]byte {
	if f.store.Load() == nil {
		return f.legacy.GetProposerHead()
	}
	return f.head
}

// CachedHeadRoot returns the published head after the fork.
func (f *ForkChoice) CachedHeadRoot() [32]byte {
	if f.store.Load() == nil {
		return f.legacy.CachedHeadRoot()
	}
	return f.head
}

// AncestorRoot returns root's ancestor at or before slot.
func (f *ForkChoice) AncestorRoot(ctx context.Context, root [32]byte, slot primitives.Slot) ([32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.AncestorRoot(ctx, root, slot)
	}
	return f.ancestor(ctx, st, root, slot)
}

// AncestorRoots returns the blocks after terminalRoot up to root, oldest first, nil if terminalRoot is not an ancestor.
func (f *ForkChoice) AncestorRoots(root, terminalRoot [32]byte) ([][32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.AncestorRoots(root, terminalRoot)
	}
	n, t := st.nodes[root], st.nodes[terminalRoot]
	if n == nil || t == nil {
		return nil, doublylinkedtree.ErrNilNode
	}
	var roots [][32]byte
	for ; n.Slot > t.Slot && n.Parent != nil; n = n.Parent {
		roots = append(roots, n.Root)
		if n.Parent == t {
			slices.Reverse(roots)
			return roots, nil
		}
	}
	return nil, nil
}

// CommonAncestor returns the latest block on both chains.
func (f *ForkChoice) CommonAncestor(ctx context.Context, root1, root2 [32]byte) ([32]byte, primitives.Slot, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.CommonAncestor(ctx, root1, root2)
	}
	a, b := st.nodes[root1], st.nodes[root2]
	if a == nil || b == nil {
		return [32]byte{}, 0, forkchoice.ErrUnknownCommonAncestor
	}
	c := commonAncestor(a, b)
	return c.Root, c.Slot, nil
}

// ForkChoiceDump fails after the fork, the debug endpoint serves the Goldilocks dump instead.
func (f *ForkChoice) ForkChoiceDump(ctx context.Context) (*forkchoice2.Dump, error) {
	if f.store.Load() == nil {
		return f.legacy.ForkChoiceDump(ctx)
	}
	return nil, errNoDump
}

// ForkChoiceDumpV2 fails after the fork, the debug endpoint serves the Goldilocks dump instead.
func (f *ForkChoice) ForkChoiceDumpV2(ctx context.Context) (*forkchoice2.DumpV2, error) {
	if f.store.Load() == nil {
		return f.legacy.ForkChoiceDumpV2(ctx)
	}
	return nil, errNoDump
}

// IsAncestor reports whether ancestorRoot is root or one of its ancestors.
func (f *ForkChoice) IsAncestor(root, ancestorRoot [32]byte) (bool, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.IsAncestor(root, ancestorRoot)
	}
	if root == ancestorRoot {
		return true, nil
	}
	n, a := st.nodes[root], st.nodes[ancestorRoot]
	if n == nil || a == nil {
		return false, doublylinkedtree.ErrNilNode
	}
	return isAncestor(a, n), nil
}

// SlashedIndices is empty after the fork.
func (f *ForkChoice) SlashedIndices() map[primitives.ValidatorIndex]bool {
	if f.store.Load() == nil {
		return f.legacy.SlashedIndices()
	}
	return map[primitives.ValidatorIndex]bool{}
}

// Tips returns the blocks without children and their slots.
func (f *ForkChoice) Tips() ([][32]byte, []primitives.Slot) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.Tips()
	}
	var roots [][32]byte
	var tipSlots []primitives.Slot
	for r, n := range st.nodes {
		if len(n.Children) == 0 {
			roots = append(roots, r)
			tipSlots = append(tipSlots, n.Slot)
		}
	}
	return roots, tipSlots
}

// UnrealizedJustification returns J after the fork.
func (f *ForkChoice) UnrealizedJustification(root [32]byte) (*forkchoicetypes.Checkpoint, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.UnrealizedJustification(root)
	}
	return checkpoint(st.justified), nil
}

// VoteSnapshot returns buf unchanged after the fork.
func (f *ForkChoice) VoteSnapshot(buf []forkchoicetypes.VoteData) []forkchoicetypes.VoteData {
	if f.store.Load() == nil {
		return f.legacy.VoteSnapshot(buf)
	}
	return buf
}

// VotingSource returns J after the fork.
func (f *ForkChoice) VotingSource(root [32]byte) (*forkchoicetypes.Checkpoint, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.VotingSource(root)
	}
	return checkpoint(st.justified), nil
}

// FinalizedCheckpoint returns F with the epoch of its slot after the fork.
func (f *ForkChoice) FinalizedCheckpoint() *forkchoicetypes.Checkpoint {
	st := f.store.Load()
	if st == nil {
		return f.legacy.FinalizedCheckpoint()
	}
	return checkpoint(st.finalized)
}

// FinalizedPayloadBlockHash returns the payload F builds on.
func (f *ForkChoice) FinalizedPayloadBlockHash() [32]byte {
	st := f.store.Load()
	if st == nil {
		return f.legacy.FinalizedPayloadBlockHash()
	}
	return f.checkpointHash(st.finalized.Root)
}

// HasFullNode reports whether root's payload arrived.
func (f *ForkChoice) HasFullNode(root [32]byte) bool {
	if f.store.Load() == nil {
		return f.legacy.HasFullNode(root)
	}
	p := f.exec[root]
	return p != nil && p.arrived
}

// HasNode reports whether root is in fork choice.
func (f *ForkChoice) HasNode(root [32]byte) bool {
	st := f.store.Load()
	if st == nil {
		return f.legacy.HasNode(root)
	}
	return st.HasNode(root)
}

// FullBeatsEmpty is true for every known block after the fork.
func (f *ForkChoice) FullBeatsEmpty(root [32]byte) bool {
	st := f.store.Load()
	if st == nil {
		return f.legacy.FullBeatsEmpty(root)
	}
	// TODO(DC): full or empty from PTC and payload arrival.
	return st.HasNode(root)
}

// HighestReceivedBlockSlot returns the slot of the highest inserted block.
func (f *ForkChoice) HighestReceivedBlockSlot() primitives.Slot {
	if f.store.Load() == nil {
		return f.legacy.HighestReceivedBlockSlot()
	}
	return f.highestSlot
}

// HighestReceivedBlockRoot returns the root of the highest inserted block.
func (f *ForkChoice) HighestReceivedBlockRoot() [32]byte {
	if f.store.Load() == nil {
		return f.legacy.HighestReceivedBlockRoot()
	}
	return f.highestRoot
}

// IsCanonical reports whether root is on the published head's chain.
func (f *ForkChoice) IsCanonical(root [32]byte) bool {
	st := f.store.Load()
	if st == nil {
		return f.legacy.IsCanonical(root)
	}
	n := st.nodes[root]
	return n != nil && isAncestor(n, st.nodes[f.head])
}

// IsOptimistic is false for every known block after the fork.
func (f *ForkChoice) IsOptimistic(root [32]byte) (bool, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.IsOptimistic(root)
	}
	if !st.HasNode(root) {
		return true, doublylinkedtree.ErrNilNode
	}
	// TODO(DC): optimistic status.
	return false, nil
}

// IsViableForCheckpoint reports whether the checkpoint block is in fork choice after the fork.
func (f *ForkChoice) IsViableForCheckpoint(cp *forkchoicetypes.Checkpoint) (bool, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.IsViableForCheckpoint(cp)
	}
	return st.HasNode(cp.Root), nil
}

// JustifiedCheckpoint returns J with the epoch of its slot after the fork.
func (f *ForkChoice) JustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	st := f.store.Load()
	if st == nil {
		return f.legacy.JustifiedCheckpoint()
	}
	return checkpoint(st.justified)
}

// JustifiedPayloadBlockHash returns the payload J builds on.
func (f *ForkChoice) JustifiedPayloadBlockHash() [32]byte {
	st := f.store.Load()
	if st == nil {
		return f.legacy.JustifiedPayloadBlockHash()
	}
	return f.checkpointHash(st.justified.Root)
}

// NodeCount returns the number of blocks in fork choice.
func (f *ForkChoice) NodeCount() int {
	st := f.store.Load()
	if st == nil {
		return f.legacy.NodeCount()
	}
	return len(st.nodes)
}

// UnrealizedJustifiedCheckpoint returns J after the fork.
func (f *ForkChoice) UnrealizedJustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	st := f.store.Load()
	if st == nil {
		return f.legacy.UnrealizedJustifiedCheckpoint()
	}
	return checkpoint(st.justified)
}

// PreviousJustifiedCheckpoint returns J after the fork.
func (f *ForkChoice) PreviousJustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	st := f.store.Load()
	if st == nil {
		return f.legacy.PreviousJustifiedCheckpoint()
	}
	return checkpoint(st.justified)
}

// ProposerBoost is zero after the fork.
func (f *ForkChoice) ProposerBoost() [32]byte {
	if f.store.Load() == nil {
		return f.legacy.ProposerBoost()
	}
	return [32]byte{}
}

// ReceivedBlocksLastEpoch counts the slots of the last epoch that have a block in fork choice.
func (f *ForkChoice) ReceivedBlocksLastEpoch() (uint64, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.ReceivedBlocksLastEpoch()
	}
	current := st.slotAt(prysmTime.Now())
	var lower primitives.Slot
	if spe := params.BeaconConfig().SlotsPerEpoch; current > spe {
		lower = current - spe
	}
	seen := make(map[primitives.Slot]bool)
	for _, n := range st.nodes {
		if n.Slot != 0 && n.Slot >= lower {
			seen[n.Slot] = true
		}
	}
	return uint64(len(seen)), nil
}

// ShouldOverrideFCU is false after the fork.
func (f *ForkChoice) ShouldOverrideFCU() bool {
	if f.store.Load() == nil {
		return f.legacy.ShouldOverrideFCU()
	}
	return false
}

// Slot returns root's slot.
func (f *ForkChoice) Slot(root [32]byte) (primitives.Slot, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.Slot(root)
	}
	n := st.nodes[root]
	if n == nil {
		return 0, doublylinkedtree.ErrNilNode
	}
	return n.Slot, nil
}

// DependentRoot returns the published head's dependent root for epoch.
func (f *ForkChoice) DependentRoot(epoch primitives.Epoch) ([32]byte, error) {
	if f.store.Load() == nil {
		return f.legacy.DependentRoot(epoch)
	}
	return f.DependentRootForEpoch(f.head, epoch)
}

// DependentRootForEpoch returns root's ancestor at the last slot before epoch, zero for epoch 0.
func (f *ForkChoice) DependentRootForEpoch(root [32]byte, epoch primitives.Epoch) ([32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.DependentRootForEpoch(root, epoch)
	}
	start, err := slots.EpochStart(epoch)
	if err != nil || start == 0 {
		return [32]byte{}, err
	}
	return f.ancestor(context.Background(), st, root, start-1)
}

// TargetRootForEpoch returns root's ancestor at epoch's first slot.
func (f *ForkChoice) TargetRootForEpoch(root [32]byte, epoch primitives.Epoch) ([32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.TargetRootForEpoch(root, epoch)
	}
	start, err := slots.EpochStart(epoch)
	if err != nil {
		return [32]byte{}, err
	}
	return f.ancestor(context.Background(), st, root, start)
}

// UnrealizedJustifiedPayloadBlockHash returns the payload J builds on after the fork.
func (f *ForkChoice) UnrealizedJustifiedPayloadBlockHash() [32]byte {
	st := f.store.Load()
	if st == nil {
		return f.legacy.UnrealizedJustifiedPayloadBlockHash()
	}
	return f.checkpointHash(st.justified.Root)
}

// Weight is zero after the fork.
func (f *ForkChoice) Weight(root [32]byte) (uint64, error) {
	if f.store.Load() == nil {
		return f.legacy.Weight(root)
	}
	return 0, nil
}

// ConsensusNodeWeight is zero after the fork.
func (f *ForkChoice) ConsensusNodeWeight(root [32]byte) (uint64, error) {
	if f.store.Load() == nil {
		return f.legacy.ConsensusNodeWeight(root)
	}
	return 0, nil
}

// CouldBuilderWithhold is false after the fork.
func (f *ForkChoice) CouldBuilderWithhold(root [32]byte) bool {
	if f.store.Load() == nil {
		return f.legacy.CouldBuilderWithhold(root)
	}
	// TODO(DC): PTC votes.
	return false
}

// BuilderIndex returns the builder of root's bid.
func (f *ForkChoice) BuilderIndex(root [32]byte) (primitives.BuilderIndex, error) {
	if f.store.Load() == nil {
		return f.legacy.BuilderIndex(root)
	}
	p := f.exec[root]
	if p == nil {
		return 0, doublylinkedtree.ErrNilNode
	}
	return p.builder, nil
}

// PayloadWeights is zero after the fork.
func (f *ForkChoice) PayloadWeights(root [32]byte) (uint64, uint64, error) {
	if f.store.Load() == nil {
		return f.legacy.PayloadWeights(root)
	}
	return 0, 0, nil
}

// HasPayloadBlockHash reports whether blockHash is root's arrived payload or the payload root builds on.
func (f *ForkChoice) HasPayloadBlockHash(root, blockHash [32]byte) bool {
	if f.store.Load() == nil {
		return f.legacy.HasPayloadBlockHash(root, blockHash)
	}
	p := f.exec[root]
	return p != nil && (blockHash == p.blockHash && p.arrived || blockHash == p.parentHash)
}

// PTCVotedEarlyAndAvailable is false after the fork.
func (f *ForkChoice) PTCVotedEarlyAndAvailable(root [32]byte) bool {
	if f.store.Load() == nil {
		return f.legacy.PTCVotedEarlyAndAvailable(root)
	}
	// TODO(DC): PTC votes.
	return false
}

// PTCVotedLate is false after the fork.
func (f *ForkChoice) PTCVotedLate(root [32]byte) bool {
	if f.store.Load() == nil {
		return f.legacy.PTCVotedLate(root)
	}
	// TODO(DC): PTC votes.
	return false
}

// ParentRoot returns root's parent, zero for F.
func (f *ForkChoice) ParentRoot(root [32]byte) ([32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.ParentRoot(root)
	}
	n := st.nodes[root]
	if n == nil {
		return [32]byte{}, doublylinkedtree.ErrNilNode
	}
	if n.Parent == nil {
		return [32]byte{}, nil
	}
	return n.Parent.Root, nil
}

// ParentHash returns the payload root builds on.
func (f *ForkChoice) ParentHash(root [32]byte) [32]byte {
	if f.store.Load() == nil {
		return f.legacy.ParentHash(root)
	}
	if p := f.exec[root]; p != nil {
		return p.parentHash
	}
	return [32]byte{}
}

// BlockHash returns root's committed payload hash.
func (f *ForkChoice) BlockHash(root [32]byte) ([32]byte, error) {
	if f.store.Load() == nil {
		return f.legacy.BlockHash(root)
	}
	p := f.exec[root]
	if p == nil {
		return [32]byte{}, doublylinkedtree.ErrNilNode
	}
	return p.blockHash, nil
}

// GasLimit returns the gas limit of root's arrived payload, or of the payload root builds on.
func (f *ForkChoice) GasLimit(root, blockHash [32]byte) (uint64, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.GasLimit(root, blockHash)
	}
	p := f.exec[root]
	if p == nil {
		return 0, doublylinkedtree.ErrNilNode
	}
	if blockHash == p.blockHash {
		if !p.arrived {
			return 0, errors.New("payload for root has not been imported")
		}
		return p.gasLimit, nil
	}
	if blockHash != p.parentHash {
		return 0, errors.New("block hash is neither the root's payload nor its parent payload")
	}
	for n := st.nodes[root].Parent; n != nil; n = n.Parent {
		if q := f.exec[n.Root]; q != nil && q.blockHash == blockHash && q.arrived {
			return q.gasLimit, nil
		}
	}
	return 0, errors.New("no full ancestor with gas limit")
}

// CanonicalNodeAtSlot returns the published head's ancestor at or before slot.
func (f *ForkChoice) CanonicalNodeAtSlot(slot primitives.Slot) ([32]byte, bool) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.CanonicalNodeAtSlot(slot)
	}
	r, err := f.ancestor(context.Background(), st, f.head, slot)
	if err != nil {
		return [32]byte{}, false
	}
	// TODO(DC): full or empty from PTC and payload arrival.
	return r, true
}

// ConfirmedPayloadBlockHash returns the payload root builds on, or root's own before Gloas.
func (f *ForkChoice) ConfirmedPayloadBlockHash(root [32]byte) [32]byte {
	if f.store.Load() == nil {
		return f.legacy.ConfirmedPayloadBlockHash(root)
	}
	return f.checkpointHash(root)
}
