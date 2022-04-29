package state_native

import (
	"context"
	"testing"

	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_RotateAttestations(t *testing.T) {
	st, err := InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot:                      1,
		CurrentEpochAttestations:  []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 456}}},
		PreviousEpochAttestations: []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 123}}},
	})
	require.NoError(t, err)

	require.NoError(t, st.RotateAttestations())
	s, ok := st.(*BeaconState)
	require.Equal(t, true, ok)
	require.Equal(t, 0, len(s.currentEpochAttestationsVal()))
	require.Equal(t, types.Slot(456), s.previousEpochAttestationsVal()[0].Data.Slot)
}

func TestAppendBeyondIndicesLimit(t *testing.T) {
	zeroHash := params.BeaconConfig().ZeroHash
	mockblockRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(mockblockRoots); i++ {
		mockblockRoots[i] = zeroHash[:]
	}

	mockstateRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(mockstateRoots); i++ {
		mockstateRoots[i] = zeroHash[:]
	}
	mockrandaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(mockrandaoMixes); i++ {
		mockrandaoMixes[i] = zeroHash[:]
	}
	newState, err := InitializeFromProtoPhase0(&ethpb.BeaconState{
		Slot:                      1,
		CurrentEpochAttestations:  []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 456}}},
		PreviousEpochAttestations: []*ethpb.PendingAttestation{{Data: &ethpb.AttestationData{Slot: 123}}},
		Validators:                []*ethpb.Validator{},
		Eth1Data:                  &ethpb.Eth1Data{},
		BlockRoots:                mockblockRoots,
		StateRoots:                mockstateRoots,
		RandaoMixes:               mockrandaoMixes,
	})
	require.NoError(t, err)
	_, err = newState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	st, ok := newState.(*BeaconState)
	require.Equal(t, true, ok)
	for i := nativetypes.FieldIndex(0); i < nativetypes.FieldIndex(params.BeaconConfig().BeaconStateFieldCount); i++ {
		st.dirtyFields[i] = true
	}
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		assert.NoError(t, st.AppendValidator(&ethpb.Validator{}))
	}
	assert.Equal(t, false, st.rebuildTrie[nativetypes.Validators])
	assert.NotEqual(t, len(st.dirtyIndices[nativetypes.Validators]), 0)

	for i := 0; i < indicesLimit; i++ {
		assert.NoError(t, st.AppendValidator(&ethpb.Validator{}))
	}
	assert.Equal(t, true, st.rebuildTrie[nativetypes.Validators])
	assert.Equal(t, len(st.dirtyIndices[nativetypes.Validators]), 0)
}
