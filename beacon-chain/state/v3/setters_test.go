package v3

import (
	"context"
	"testing"

	stateTypes "github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

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
	st, err := InitializeFromProto(&ethpb.BeaconStateAltair{
		Slot:                       1,
		CurrentEpochParticipation:  []byte{},
		PreviousEpochParticipation: []byte{},
		Validators:                 []*eth.Validator{},
		Eth1Data:                   &eth.Eth1Data{},
		BlockRoots:                 mockblockRoots,
		StateRoots:                 mockstateRoots,
		RandaoMixes:                mockrandaoMixes,
	})
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	for i := stateTypes.FieldIndex(0); i < stateTypes.FieldIndex(params.BeaconConfig().BeaconStateAltairFieldCount); i++ {
		st.dirtyFields[i] = true
	}
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		assert.NoError(t, st.AppendValidator(&eth.Validator{}))
	}
	assert.Equal(t, false, st.rebuildTrie[validators])
	assert.NotEqual(t, len(st.dirtyIndices[validators]), 0)

	for i := 0; i < indicesLimit; i++ {
		assert.NoError(t, st.AppendValidator(&eth.Validator{}))
	}
	assert.Equal(t, true, st.rebuildTrie[validators])
	assert.Equal(t, len(st.dirtyIndices[validators]), 0)
}
