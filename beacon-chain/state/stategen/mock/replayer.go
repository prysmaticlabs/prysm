package mock

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

func NewMockReplayerBuilder(opt ...MockReplayerBuilderOption) *MockReplayerBuilder {
	b := &MockReplayerBuilder{}
	for _, o := range opt {
		o(b)
	}
	return b
}

type MockReplayerBuilderOption func(*MockReplayerBuilder)

func WithMockState(s state.BeaconState) MockReplayerBuilderOption {
	return func(b *MockReplayerBuilder) {
		b.SetMockState(s)
	}
}

func WithStateError(s types.Slot, e error) MockReplayerBuilderOption {
	return func(b *MockReplayerBuilder) {
		b.SetMockSlotError(s, e)
	}
}

type MockReplayerBuilder struct {
	forSlot map[types.Slot]*MockReplayer
}

func (b *MockReplayerBuilder) SetMockState(s state.BeaconState) {
	if b.forSlot == nil {
		b.forSlot = make(map[types.Slot]*MockReplayer)
	}
	b.forSlot[s.Slot()] = &MockReplayer{State: s}
}

func (b *MockReplayerBuilder) SetMockStateForSlot(s state.BeaconState, slot types.Slot) {
	if b.forSlot == nil {
		b.forSlot = make(map[types.Slot]*MockReplayer)
	}
	b.forSlot[slot] = &MockReplayer{State: s}
}

func (b *MockReplayerBuilder) SetMockSlotError(s types.Slot, e error) {
	if b.forSlot == nil {
		b.forSlot = make(map[types.Slot]*MockReplayer)
	}
	b.forSlot[s] = &MockReplayer{Err: e}
}

func (b *MockReplayerBuilder) ReplayerForSlot(target types.Slot) stategen.Replayer {
	return b.forSlot[target]
}

var _ stategen.ReplayerBuilder = &MockReplayerBuilder{}

type MockReplayer struct {
	State state.BeaconState
	Err   error
}

func (m *MockReplayer) ReplayBlocks(_ context.Context) (state.BeaconState, error) {
	return m.State, m.Err
}

func (m *MockReplayer) ReplayToSlot(_ context.Context, _ types.Slot) (state.BeaconState, error) {
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
	Slot types.Slot
}

func (c *MockCurrentSlotter) CurrentSlot() types.Slot {
	return c.Slot
}

var _ stategen.CurrentSlotter = &MockCurrentSlotter{}
