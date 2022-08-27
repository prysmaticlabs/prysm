package mock

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
)

// NewMockSaver creates a value that can be used as a stategen.Saver in tests
func NewMockSaver(d iface.Database) *MockSaver {
	return &MockSaver{
		db: d,
	}
}

type MockSaver struct {
	db iface.HeadAccessDatabase
}

// Save just saves everything it is asked to save
func (s *MockSaver) Save(ctx context.Context, root [32]byte, st state.BeaconState) error {
	return s.db.SaveState(ctx, st, root)
}

// Preserve just checks to see if the state has been saved before saving it
func (s *MockSaver) Preserve(ctx context.Context, root [32]byte, st state.BeaconState) error {
	if !s.db.HasState(ctx, root) {
		return s.Save(ctx, root, st)
	}
	return nil
}

var _ stategen.Saver = &MockSaver{}

// NewMockStategen hides the complexity of preparing all the values needed to construct an
// instance of stategen.State for tests that don't make assumptions about the internal behavior of stategen.
func NewMockStategen(db iface.HeadAccessDatabase, opts ...stategen.StateGenOption) *stategen.State {
	saver := &MockSaver{db: db}
	return stategen.New(db, saver, opts...)
}
