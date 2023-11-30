package forkchoice

import (
	"context"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	forkchoice2 "github.com/prysmaticlabs/prysm/v4/consensus-types/forkchoice"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type ROForkChoice struct {
	getter Getter
	l      Locker
}

var _ Getter = &ROForkChoice{}

// ROWrappable represents the subset of ForkChoicer a type needs to support
// in order for ROForkChoice to wrap it.
type ROWrappable interface {
	Locker
	Getter
}

// NewROForkChoice returns an ROForkChoice that delegates forkchoice.Getter calls to the
// given value after first using its Locker methods to make sure it is correctly locked.
func NewROForkChoice(w ROWrappable) *ROForkChoice {
	return &ROForkChoice{getter: w, l: w}
}

// HasNode delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) HasNode(root [32]byte) bool {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.HasNode(root)
}

// ProposerBoost delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) ProposerBoost() [fieldparams.RootLength]byte {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.ProposerBoost()
}

// AncestorRoot delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) AncestorRoot(ctx context.Context, root [32]byte, slot primitives.Slot) ([32]byte, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.AncestorRoot(ctx, root, slot)
}

// CommonAncestor delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) CommonAncestor(ctx context.Context, root1 [32]byte, root2 [32]byte) ([32]byte, primitives.Slot, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.CommonAncestor(ctx, root1, root2)
}

// IsCanonical delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) IsCanonical(root [32]byte) bool {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.IsCanonical(root)
}

// FinalizedCheckpoint delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) FinalizedCheckpoint() *forkchoicetypes.Checkpoint {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.FinalizedCheckpoint()
}

// IsViableForCheckpoint delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) IsViableForCheckpoint(cp *forkchoicetypes.Checkpoint) (bool, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.IsViableForCheckpoint(cp)
}

// FinalizedPayloadBlockHash delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) FinalizedPayloadBlockHash() [32]byte {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.FinalizedPayloadBlockHash()
}

// JustifiedCheckpoint delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) JustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.JustifiedCheckpoint()
}

// PreviousJustifiedCheckpoint delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) PreviousJustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.PreviousJustifiedCheckpoint()
}

// JustifiedPayloadBlockHash delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) JustifiedPayloadBlockHash() [32]byte {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.JustifiedPayloadBlockHash()
}

// UnrealizedJustifiedPayloadBlockHash delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) UnrealizedJustifiedPayloadBlockHash() [32]byte {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.UnrealizedJustifiedPayloadBlockHash()
}

// NodeCount delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) NodeCount() int {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.NodeCount()
}

// HighestReceivedBlockSlot delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) HighestReceivedBlockSlot() primitives.Slot {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.HighestReceivedBlockSlot()
}

// ReceivedBlocksLastEpoch delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) ReceivedBlocksLastEpoch() (uint64, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.ReceivedBlocksLastEpoch()
}

// ForkChoiceDump delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) ForkChoiceDump(ctx context.Context) (*forkchoice2.Dump, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.ForkChoiceDump(ctx)
}

// Weight delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) Weight(root [32]byte) (uint64, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.Weight(root)
}

// Tips delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) Tips() ([][32]byte, []primitives.Slot) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.Tips()
}

// IsOptimistic delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) IsOptimistic(root [32]byte) (bool, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.IsOptimistic(root)
}

// ShouldOverrideFCU delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) ShouldOverrideFCU() bool {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.ShouldOverrideFCU()
}

// Slot delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) Slot(root [32]byte) (primitives.Slot, error) {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.Slot(root)
}

// LastRoot delegates to the underlying forkchoice call, under a lock.
func (ro *ROForkChoice) LastRoot(e primitives.Epoch) [32]byte {
	ro.l.RLock()
	defer ro.l.RUnlock()
	return ro.getter.LastRoot(e)
}
