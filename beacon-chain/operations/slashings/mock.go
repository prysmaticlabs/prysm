package slashings

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// PoolMock is a fake implementation of PoolManager.
type PoolMock struct {
	PendingAttSlashings  []*ethpb.AttesterSlashing
	PendingPropSlashings []*ethpb.ProposerSlashing
}

// PendingAttesterSlashings --
func (m *PoolMock) PendingAttesterSlashings(ctx context.Context, state *state.BeaconState, noLimit bool) []*ethpb.AttesterSlashing {
	return m.PendingAttSlashings
}

// PendingProposerSlashings --
func (m *PoolMock) PendingProposerSlashings(ctx context.Context, state *state.BeaconState, noLimit bool) []*ethpb.ProposerSlashing {
	return m.PendingPropSlashings
}

// InsertAttesterSlashing --
func (m *PoolMock) InsertAttesterSlashing(ctx context.Context, state *state.BeaconState, slashing *ethpb.AttesterSlashing) error {
	panic("implement me")
}

// InsertProposerSlashing --
func (m *PoolMock) InsertProposerSlashing(ctx context.Context, state *state.BeaconState, slashing *ethpb.ProposerSlashing) error {
	panic("implement me")
}

// MarkIncludedAttesterSlashing --
func (m *PoolMock) MarkIncludedAttesterSlashing(as *ethpb.AttesterSlashing) {
	panic("implement me")
}

// MarkIncludedProposerSlashing --
func (m *PoolMock) MarkIncludedProposerSlashing(ps *ethpb.ProposerSlashing) {
	panic("implement me")
}
