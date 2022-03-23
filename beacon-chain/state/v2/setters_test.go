package v2

import (
	"context"
	"strconv"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	testtmpl "github.com/prysmaticlabs/prysm/beacon-chain/state/testing"
	stateTypes "github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
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
		Validators:                 []*ethpb.Validator{},
		Eth1Data:                   &ethpb.Eth1Data{},
		BlockRoots:                 mockblockRoots,
		StateRoots:                 mockstateRoots,
		RandaoMixes:                mockrandaoMixes,
	})
	require.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	s, ok := st.(*BeaconState)
	require.Equal(t, true, ok)
	for i := stateTypes.FieldIndex(0); i < stateTypes.FieldIndex(params.BeaconConfig().BeaconStateAltairFieldCount); i++ {
		s.dirtyFields[i] = true
	}
	_, err = st.HashTreeRoot(context.Background())
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		assert.NoError(t, st.AppendValidator(&ethpb.Validator{}))
	}
	assert.Equal(t, false, s.rebuildTrie[validators])
	assert.NotEqual(t, len(s.dirtyIndices[validators]), 0)

	for i := 0; i < indicesLimit; i++ {
		assert.NoError(t, st.AppendValidator(&ethpb.Validator{}))
	}
	assert.Equal(t, true, s.rebuildTrie[validators])
	assert.Equal(t, len(s.dirtyIndices[validators]), 0)
}

func TestBeaconState_AppendBalanceWithTrie(t *testing.T) {
	count := uint64(100)
	vals := make([]*ethpb.Validator, 0, count)
	bals := make([]uint64, 0, count)
	for i := uint64(1); i < count; i++ {
		someRoot := [32]byte{}
		someKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(someRoot[:], strconv.Itoa(int(i)))
		copy(someKey[:], strconv.Itoa(int(i)))
		vals = append(vals, &ethpb.Validator{
			PublicKey:                  someKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 1,
			ActivationEpoch:            1,
			ExitEpoch:                  1,
			WithdrawableEpoch:          1,
		})
		bals = append(bals, params.BeaconConfig().MaxEffectiveBalance)
	}
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
	var pubKeys [][]byte
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		pubKeys = append(pubKeys, bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength))
	}
	st, err := InitializeFromProto(&ethpb.BeaconStateAltair{
		Slot:                  1,
		GenesisValidatorsRoot: make([]byte, 32),
		Fork: &ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
			Epoch:           0,
		},
		LatestBlockHeader: &ethpb.BeaconBlockHeader{
			ParentRoot: make([]byte, fieldparams.RootLength),
			StateRoot:  make([]byte, fieldparams.RootLength),
			BodyRoot:   make([]byte, fieldparams.RootLength),
		},
		CurrentEpochParticipation:  []byte{},
		PreviousEpochParticipation: []byte{},
		Validators:                 vals,
		Balances:                   bals,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, fieldparams.RootLength),
			BlockHash:   make([]byte, 32),
		},
		BlockRoots:                  mockblockRoots,
		StateRoots:                  mockstateRoots,
		RandaoMixes:                 mockrandaoMixes,
		JustificationBits:           bitfield.NewBitvector4(),
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		CurrentSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: make([]byte, 48),
		},
		NextSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: make([]byte, 48),
		},
	})
	assert.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			assert.NoError(t, st.UpdateBalancesAtIndex(types.ValidatorIndex(i), 1000))
		}
		if i%3 == 0 {
			assert.NoError(t, st.AppendBalance(1000))
		}
	}
	_, err = st.HashTreeRoot(context.Background())
	assert.NoError(t, err)
	s, ok := st.(*BeaconState)
	require.Equal(t, true, ok)
	newRt := bytesutil.ToBytes32(s.merkleLayers[0][balances])
	wantedRt, err := stateutil.Uint64ListRootWithRegistryLimit(s.state.Balances)
	assert.NoError(t, err)
	assert.Equal(t, wantedRt, newRt, "state roots are unequal")
}

func TestBeaconState_ModifyPreviousParticipationBits(t *testing.T) {
	testState := createState(200)
	testtmpl.VerifyBeaconStateModifyPreviousParticipationField(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto(testState)
		},
	)
	testtmpl.VerifyBeaconStateModifyPreviousParticipationField_NestedAction(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto(testState)
		},
	)
}

func TestBeaconState_ModifyCurrentParticipationBits(t *testing.T) {
	testState := createState(200)
	testtmpl.VerifyBeaconStateModifyCurrentParticipationField(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto(testState)
		},
	)
	testtmpl.VerifyBeaconStateModifyCurrentParticipationField_NestedAction(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto(testState)
		},
	)
}

func createState(count uint64) *ethpb.BeaconStateAltair {
	vals := make([]*ethpb.Validator, 0, count)
	bals := make([]uint64, 0, count)
	for i := uint64(0); i < count; i++ {
		someRoot := [32]byte{}
		someKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(someRoot[:], strconv.Itoa(int(i)))
		copy(someKey[:], strconv.Itoa(int(i)))
		vals = append(vals, &ethpb.Validator{
			PublicKey:                  someKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 1,
			ActivationEpoch:            1,
			ExitEpoch:                  1,
			WithdrawableEpoch:          1,
		})
		bals = append(bals, params.BeaconConfig().MaxEffectiveBalance)
	}
	return &ethpb.BeaconStateAltair{
		CurrentEpochParticipation:  make([]byte, count),
		PreviousEpochParticipation: make([]byte, count),
		Validators:                 vals,
		Balances:                   bals,
	}
}
