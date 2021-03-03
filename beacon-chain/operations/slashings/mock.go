package slashings

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
)

// PoolMock is a fake implementation of PoolManager.
type PoolMock struct {
	PendingAttSlashings  []*ethpb.AttesterSlashing
	PendingPropSlashings []*ethpb.ProposerSlashing
}

// PendingAttesterSlashings --
func (m *PoolMock) PendingAttesterSlashings(_ context.Context, _ iface.ReadOnlyBeaconState, _ bool) []*ethpb.AttesterSlashing {
	return m.PendingAttSlashings
}

// PendingProposerSlashings --
func (m *PoolMock) PendingProposerSlashings(_ context.Context, _ iface.ReadOnlyBeaconState, _ bool) []*ethpb.ProposerSlashing {
	return m.PendingPropSlashings
}

// InsertAttesterSlashing --
func (m *PoolMock) InsertAttesterSlashing(_ context.Context, _ iface.ReadOnlyBeaconState, slashing *ethpb.AttesterSlashing) error {
	m.PendingAttSlashings = append(m.PendingAttSlashings, slashing)
	return nil
}

// InsertProposerSlashing --
func (m *PoolMock) InsertProposerSlashing(_ context.Context, _ iface.BeaconState, slashing *ethpb.ProposerSlashing) error {
	m.PendingPropSlashings = append(m.PendingPropSlashings, slashing)
	return nil
}

// MarkIncludedAttesterSlashing --
func (m *PoolMock) MarkIncludedAttesterSlashing(_ *ethpb.AttesterSlashing) {
	panic("implement me")
}

// MarkIncludedProposerSlashing --
func (m *PoolMock) MarkIncludedProposerSlashing(_ *ethpb.ProposerSlashing) {
	panic("implement me")
}
