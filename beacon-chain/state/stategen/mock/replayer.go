package mock

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
)

func NewMockReplayerBuilder(opt ...MockReplayerBuilderOption) *MockReplayerBuilder {
	b := &MockReplayerBuilder{}
	for _, o := range opt {
		o(b)
	}
	return b
}

type MockReplayerBuilderOption func(*MockReplayerBuilder)

func WithMockState(s types.BeaconState) MockReplayerBuilderOption {
	return func(b *MockReplayerBuilder) {
		b.SetMockState(s)
	}
}

type MockReplayerBuilder struct {
	forSlot map[primitives.Slot]*MockReplayer
}

func (b *MockReplayerBuilder) SetMockState(s types.BeaconState) {
	if b.forSlot == nil {
		b.forSlot = make(map[primitives.Slot]*MockReplayer)
	}
	b.forSlot[s.Slot()] = &MockReplayer{State: s}
}

func (b *MockReplayerBuilder) SetMockStateForSlot(s types.BeaconState, slot primitives.Slot) {
	if b.forSlot == nil {
		b.forSlot = make(map[primitives.Slot]*MockReplayer)
	}
	b.forSlot[slot] = &MockReplayer{State: s}
}

func (b *MockReplayerBuilder) SetMockSlotError(s primitives.Slot, e error) {
	if b.forSlot == nil {
		b.forSlot = make(map[primitives.Slot]*MockReplayer)
	}
	b.forSlot[s] = &MockReplayer{Err: e}
}

func (b *MockReplayerBuilder) ReplayerForSlot(target primitives.Slot) stategen.Replayer {
	return b.forSlot[target]
}

var _ stategen.ReplayerBuilder = &MockReplayerBuilder{}

type MockReplayer struct {
	State types.BeaconState
	Err   error
}

func (m *MockReplayer) ReplayBlocks(_ context.Context) (types.BeaconState, error) {
	return m.State, m.Err
}

func (m *MockReplayer) ReplayToSlot(_ context.Context, _ primitives.Slot) (types.BeaconState, error) {
	return m.State, m.Err
}

var _ stategen.Replayer = &MockReplayer{}

type MockCanonicalChecker struct {
	Is  bool
	Err error
}

func (m *MockCanonicalChecker) IsCanonical(_ context.Context, _ [32]byte) (bool, error) {
	return m.Is, m.Err
}

var _ stategen.CanonicalChecker = &MockCanonicalChecker{}

type MockCurrentSlotter struct {
	Slot primitives.Slot
}

func (c *MockCurrentSlotter) CurrentSlot() primitives.Slot {
	return c.Slot
}

var _ stategen.CurrentSlotter = &MockCurrentSlotter{}
