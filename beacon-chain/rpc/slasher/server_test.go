package slasher

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/slasher"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_IsSlashableAttestation_SlashingFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		AttesterSlashingFound: true,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableAttestation(ctx, &ethpb.IndexedAttestation{})
	require.NoError(t, err)
	require.NotNil(t, slashing.AttesterSlashing)
}

func TestServer_IsSlashableAttestation_SlashingNotFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		AttesterSlashingFound: false,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableAttestation(ctx, &ethpb.IndexedAttestation{})
	require.NoError(t, err)
	require.Equal(t, (*ethpb.AttesterSlashing)(nil), slashing.AttesterSlashing)
}

func TestServer_IsSlashableBlock_SlashingFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		ProposerSlashingFound: true,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableBlock(ctx, &ethpb.SignedBeaconBlockHeader{})
	require.NoError(t, err)
	require.NotNil(t, slashing.ProposerSlashing)
}

func TestServer_IsSlashableBlock_SlashingNotFound(t *testing.T) {
	mockSlasher := &slasher.MockSlashingChecker{
		ProposerSlashingFound: false,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	slashing, err := s.IsSlashableBlock(ctx, &ethpb.SignedBeaconBlockHeader{})
	require.NoError(t, err)
	require.Equal(t, (*ethpb.ProposerSlashing)(nil), slashing.ProposerSlashing)
}
