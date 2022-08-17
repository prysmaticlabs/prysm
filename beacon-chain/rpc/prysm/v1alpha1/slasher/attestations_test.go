package slasher

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/slasher/mock"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestServer_HighestAttestations(t *testing.T) {
	highestAtts := map[types.ValidatorIndex]*ethpb.HighestAttestation{
		0: {
			ValidatorIndex:     0,
			HighestSourceEpoch: 1,
			HighestTargetEpoch: 2,
		},
		1: {
			ValidatorIndex:     1,
			HighestSourceEpoch: 2,
			HighestTargetEpoch: 3,
		},
	}
	mockSlasher := &mock.MockSlashingChecker{
		HighestAtts: highestAtts,
	}
	s := Server{SlashingChecker: mockSlasher}
	ctx := context.Background()
	t.Run("single index found", func(t *testing.T) {
		resp, err := s.HighestAttestations(ctx, &ethpb.HighestAttestationRequest{
			ValidatorIndices: []uint64{0},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Attestations))
		require.DeepEqual(t, highestAtts[0], resp.Attestations[0])
	})
	t.Run("single index not found", func(t *testing.T) {
		resp, err := s.HighestAttestations(ctx, &ethpb.HighestAttestationRequest{
			ValidatorIndices: []uint64{3},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(resp.Attestations))
	})
	t.Run("multiple indices all found", func(t *testing.T) {
		resp, err := s.HighestAttestations(ctx, &ethpb.HighestAttestationRequest{
			ValidatorIndices: []uint64{0, 1},
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(resp.Attestations))
		require.DeepEqual(t, highestAtts[0], resp.Attestations[0])
		require.DeepEqual(t, highestAtts[1], resp.Attestations[1])
	})
	t.Run("multiple indices some not found", func(t *testing.T) {
		resp, err := s.HighestAttestations(ctx, &ethpb.HighestAttestationRequest{
			ValidatorIndices: []uint64{0, 3},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Attestations))
		require.DeepEqual(t, highestAtts[0], resp.Attestations[0])
	})
}
