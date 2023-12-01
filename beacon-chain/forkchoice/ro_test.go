package forkchoice

import (
	"context"
	"testing"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	forkchoice2 "github.com/prysmaticlabs/prysm/v4/consensus-types/forkchoice"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

type mockCall int

const (
	lockCalled mockCall = iota
	unlockCalled
	rlockCalled
	runlockCalled
	hasNodeCalled
	proposerBoostCalled
	ancestorRootCalled
	commonAncestorCalled
	isCanonicalCalled
	finalizedCheckpointCalled
	isViableForCheckpointCalled
	finalizedPayloadBlockHashCalled
	justifiedCheckpointCalled
	previousJustifiedCheckpointCalled
	justifiedPayloadBlockHashCalled
	unrealizedJustifiedPayloadBlockHashCalled
	nodeCountCalled
	highestReceivedBlockSlotCalled
	receivedBlocksLastEpochCalled
	forkChoiceDumpCalled
	weightCalled
	tipsCalled
	isOptimisticCalled
	shouldOverrideFCUCalled
	slotCalled
	lastRootCalled
)

// ensures that the correct func was called with the correct lock pattern
// for each method in the interface.
func TestROLocking(t *testing.T) {
	cases := []struct {
		name string
		call mockCall
		cb   func(Getter)
	}{
		{
			name: "hasNodeCalled",
			call: hasNodeCalled,
			cb:   func(g Getter) { g.HasNode([32]byte{}) },
		},
		{
			name: "proposerBoostCalled",
			call: proposerBoostCalled,
			cb:   func(g Getter) { g.ProposerBoost() },
		},
		{
			name: "ancestorRootCalled",
			call: ancestorRootCalled,
			cb:   func(g Getter) { g.AncestorRoot(context.TODO(), [32]byte{}, 0) },
		},
		{
			name: "commonAncestorCalled",
			call: commonAncestorCalled,
			cb:   func(g Getter) { g.CommonAncestor(context.TODO(), [32]byte{}, [32]byte{}) },
		},
		{
			name: "isCanonicalCalled",
			call: isCanonicalCalled,
			cb:   func(g Getter) { g.IsCanonical([32]byte{}) },
		},
		{
			name: "finalizedCheckpointCalled",
			call: finalizedCheckpointCalled,
			cb:   func(g Getter) { g.FinalizedCheckpoint() },
		},
		{
			name: "isViableForCheckpointCalled",
			call: isViableForCheckpointCalled,
			cb:   func(g Getter) { g.IsViableForCheckpoint(nil) },
		},
		{
			name: "finalizedPayloadBlockHashCalled",
			call: finalizedPayloadBlockHashCalled,
			cb:   func(g Getter) { g.FinalizedPayloadBlockHash() },
		},
		{
			name: "justifiedCheckpointCalled",
			call: justifiedCheckpointCalled,
			cb:   func(g Getter) { g.JustifiedCheckpoint() },
		},
		{
			name: "previousJustifiedCheckpointCalled",
			call: previousJustifiedCheckpointCalled,
			cb:   func(g Getter) { g.PreviousJustifiedCheckpoint() },
		},
		{
			name: "justifiedPayloadBlockHashCalled",
			call: justifiedPayloadBlockHashCalled,
			cb:   func(g Getter) { g.JustifiedPayloadBlockHash() },
		},
		{
			name: "unrealizedJustifiedPayloadBlockHashCalled",
			call: unrealizedJustifiedPayloadBlockHashCalled,
			cb:   func(g Getter) { g.UnrealizedJustifiedPayloadBlockHash() },
		},
		{
			name: "nodeCountCalled",
			call: nodeCountCalled,
			cb:   func(g Getter) { g.NodeCount() },
		},
		{
			name: "highestReceivedBlockSlotCalled",
			call: highestReceivedBlockSlotCalled,
			cb:   func(g Getter) { g.HighestReceivedBlockSlot() },
		},
		{
			name: "receivedBlocksLastEpochCalled",
			call: receivedBlocksLastEpochCalled,
			cb:   func(g Getter) { g.ReceivedBlocksLastEpoch() },
		},
		{
			name: "forkChoiceDumpCalled",
			call: forkChoiceDumpCalled,
			cb:   func(g Getter) { g.ForkChoiceDump(context.TODO()) },
		},
		{
			name: "weightCalled",
			call: weightCalled,
			cb:   func(g Getter) { g.Weight([32]byte{}) },
		},
		{
			name: "tipsCalled",
			call: tipsCalled,
			cb:   func(g Getter) { g.Tips() },
		},
		{
			name: "isOptimisticCalled",
			call: isOptimisticCalled,
			cb:   func(g Getter) { g.IsOptimistic([32]byte{}) },
		},
		{
			name: "shouldOverrideFCUCalled",
			call: shouldOverrideFCUCalled,
			cb:   func(g Getter) { g.ShouldOverrideFCU() },
		},
		{
			name: "slotCalled",
			call: slotCalled,
			cb:   func(g Getter) { g.Slot([32]byte{}) },
		},
		{
			name: "lastRootCalled",
			call: lastRootCalled,
			cb:   func(g Getter) { g.LastRoot(0) },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := &mockROForkchoice{}
			ro := NewROForkChoice(m)
			c.cb(ro)
			require.Equal(t, rlockCalled, m.calls[0])
			require.Equal(t, c.call, m.calls[1])
			require.Equal(t, runlockCalled, m.calls[2])
		})
	}
}

type mockROForkchoice struct {
	calls []mockCall
}

var _ Getter = &mockROForkchoice{}
var _ Locker = &mockROForkchoice{}

func (ro *mockROForkchoice) Lock() {
	ro.calls = append(ro.calls, lockCalled)
}

func (ro *mockROForkchoice) RLock() {
	ro.calls = append(ro.calls, rlockCalled)
}

func (ro *mockROForkchoice) Unlock() {
	ro.calls = append(ro.calls, unlockCalled)
}

func (ro *mockROForkchoice) RUnlock() {
	ro.calls = append(ro.calls, runlockCalled)
}

func (ro *mockROForkchoice) HasNode(root [32]byte) bool {
	ro.calls = append(ro.calls, hasNodeCalled)
	return false
}

func (ro *mockROForkchoice) ProposerBoost() [fieldparams.RootLength]byte {
	ro.calls = append(ro.calls, proposerBoostCalled)
	return [fieldparams.RootLength]byte{}
}

func (ro *mockROForkchoice) AncestorRoot(ctx context.Context, root [32]byte, slot primitives.Slot) ([32]byte, error) {
	ro.calls = append(ro.calls, ancestorRootCalled)
	return [32]byte{}, nil
}

func (ro *mockROForkchoice) CommonAncestor(ctx context.Context, root1 [32]byte, root2 [32]byte) ([32]byte, primitives.Slot, error) {
	ro.calls = append(ro.calls, commonAncestorCalled)
	return [32]byte{}, 0, nil
}

func (ro *mockROForkchoice) IsCanonical(root [32]byte) bool {
	ro.calls = append(ro.calls, isCanonicalCalled)
	return false
}

func (ro *mockROForkchoice) FinalizedCheckpoint() *forkchoicetypes.Checkpoint {
	ro.calls = append(ro.calls, finalizedCheckpointCalled)
	return nil
}

func (ro *mockROForkchoice) IsViableForCheckpoint(cp *forkchoicetypes.Checkpoint) (bool, error) {
	ro.calls = append(ro.calls, isViableForCheckpointCalled)
	return false, nil
}

func (ro *mockROForkchoice) FinalizedPayloadBlockHash() [32]byte {
	ro.calls = append(ro.calls, finalizedPayloadBlockHashCalled)
	return [32]byte{}
}

func (ro *mockROForkchoice) JustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	ro.calls = append(ro.calls, justifiedCheckpointCalled)
	return nil
}

func (ro *mockROForkchoice) PreviousJustifiedCheckpoint() *forkchoicetypes.Checkpoint {
	ro.calls = append(ro.calls, previousJustifiedCheckpointCalled)
	return nil
}

func (ro *mockROForkchoice) JustifiedPayloadBlockHash() [32]byte {
	ro.calls = append(ro.calls, justifiedPayloadBlockHashCalled)
	return [32]byte{}
}

func (ro *mockROForkchoice) UnrealizedJustifiedPayloadBlockHash() [32]byte {
	ro.calls = append(ro.calls, unrealizedJustifiedPayloadBlockHashCalled)
	return [32]byte{}
}

func (ro *mockROForkchoice) NodeCount() int {
	ro.calls = append(ro.calls, nodeCountCalled)
	return 0
}

func (ro *mockROForkchoice) HighestReceivedBlockSlot() primitives.Slot {
	ro.calls = append(ro.calls, highestReceivedBlockSlotCalled)
	return 0
}

func (ro *mockROForkchoice) ReceivedBlocksLastEpoch() (uint64, error) {
	ro.calls = append(ro.calls, receivedBlocksLastEpochCalled)
	return 0, nil
}

func (ro *mockROForkchoice) ForkChoiceDump(ctx context.Context) (*forkchoice2.Dump, error) {
	ro.calls = append(ro.calls, forkChoiceDumpCalled)
	return nil, nil
}

func (ro *mockROForkchoice) Weight(root [32]byte) (uint64, error) {
	ro.calls = append(ro.calls, weightCalled)
	return 0, nil
}

func (ro *mockROForkchoice) Tips() ([][32]byte, []primitives.Slot) {
	ro.calls = append(ro.calls, tipsCalled)
	return make([][32]byte, 0), []primitives.Slot{}
}

func (ro *mockROForkchoice) IsOptimistic(root [32]byte) (bool, error) {
	ro.calls = append(ro.calls, isOptimisticCalled)
	return false, nil
}

func (ro *mockROForkchoice) ShouldOverrideFCU() bool {
	ro.calls = append(ro.calls, shouldOverrideFCUCalled)
	return false
}

func (ro *mockROForkchoice) Slot(root [32]byte) (primitives.Slot, error) {
	ro.calls = append(ro.calls, slotCalled)
	return 0, nil
}

func (ro *mockROForkchoice) LastRoot(e primitives.Epoch) [32]byte {
	ro.calls = append(ro.calls, lastRootCalled)
	return [32]byte{}
}
