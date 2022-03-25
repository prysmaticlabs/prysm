package testing

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"

	"testing"
)

func VerifyBeaconStateJustificationBitsNil(t *testing.T, factory getState) {
	s, err := factory()
	require.NoError(t, err)

	bits := s.JustificationBits()
	require.DeepEqual(t, bitfield.Bitvector4{}.Bytes(), bits.Bytes())
}

type getStateWithJustificationBits = func(bitfield.Bitvector4) (state.BeaconState, error)

func VerifyBeaconStateJustificationBits(t *testing.T, factory getStateWithJustificationBits) {
	s, err := factory(bitfield.Bitvector4{1, 2, 3, 4})
	require.NoError(t, err)

	bits := s.JustificationBits()
	require.DeepEqual(t, bitfield.Bitvector4{1, 2, 3, 4}.Bytes(), bits.Bytes())
}

func VerifyBeaconStatePreviousJustifiedCheckpointNil(t *testing.T, factory getState) {
	s, err := factory()

	require.NoError(t, err)

	checkpoint := s.PreviousJustifiedCheckpoint()
	require.Equal(t, (*ethpb.Checkpoint)(nil), checkpoint)
}

type getStateWithCheckpoint = func(checkpoint *ethpb.Checkpoint) (state.BeaconState, error)

func VerifyBeaconStatePreviousJustifiedCheckpoint(t *testing.T, factory getStateWithCheckpoint) {
	orgCheckpoint := &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)}
	orgCheckpoint.Root[1] = 1
	orgCheckpoint.Root[2] = 2
	orgCheckpoint.Root[3] = 3
	s, err := factory(orgCheckpoint)

	require.NoError(t, err)

	checkpoint := s.PreviousJustifiedCheckpoint()
	require.DeepEqual(t, orgCheckpoint.Root, checkpoint.Root)
}

func VerifyBeaconStateCurrentJustifiedCheckpointNil(t *testing.T, factory getState) {
	s, err := factory()

	require.NoError(t, err)

	checkpoint := s.CurrentJustifiedCheckpoint()
	require.Equal(t, (*ethpb.Checkpoint)(nil), checkpoint)
}

func VerifyBeaconStateCurrentJustifiedCheckpoint(t *testing.T, factory getStateWithCheckpoint) {
	orgCheckpoint := &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)}
	orgCheckpoint.Root[1] = 1
	orgCheckpoint.Root[2] = 2
	orgCheckpoint.Root[3] = 3
	s, err := factory(orgCheckpoint)

	require.NoError(t, err)

	checkpoint := s.CurrentJustifiedCheckpoint()
	require.DeepEqual(t, orgCheckpoint.Root, checkpoint.Root)
}

func VerifyBeaconStateFinalizedCheckpointNil(t *testing.T, factory getState) {
	s, err := factory()

	require.NoError(t, err)

	checkpoint := s.FinalizedCheckpoint()
	require.Equal(t, (*ethpb.Checkpoint)(nil), checkpoint)
	epoch := s.FinalizedCheckpointEpoch()
	require.Equal(t, types.Epoch(0), epoch)
}

func VerifyBeaconStateFinalizedCheckpoint(t *testing.T, factory getStateWithCheckpoint) {
	orgCheckpoint := &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)}
	orgCheckpoint.Root[1] = 1
	orgCheckpoint.Root[2] = 2
	orgCheckpoint.Root[3] = 3
	orgCheckpoint.Epoch = 123
	s, err := factory(orgCheckpoint)

	require.NoError(t, err)

	checkpoint := s.FinalizedCheckpoint()
	require.DeepEqual(t, orgCheckpoint.Root, checkpoint.Root)
	epoch := s.FinalizedCheckpointEpoch()
	require.Equal(t, orgCheckpoint.Epoch, epoch)
}
