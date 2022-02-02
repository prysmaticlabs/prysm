package v2

import (
	"context"
	"strconv"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestBeaconState_AppendBalanceWithTrie(t *testing.T) {
	count := uint64(100)
	vals := make([]*ethpb.Validator, 0, count)
	bals := make([]uint64, 0, count)
	for i := uint64(1); i < count; i++ {
		someRoot := [fieldparams.RootLength]byte{}
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
		GenesisValidatorsRoot: make([]byte, fieldparams.RootLength),
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
			BlockHash:   make([]byte, fieldparams.RootLength),
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
			AggregatePubkey: make([]byte, fieldparams.BLSPubkeyLength),
		},
		NextSyncCommittee: &ethpb.SyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: make([]byte, fieldparams.BLSPubkeyLength),
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
	newRt := bytesutil.ToBytes32(st.merkleLayers[0][balances])
	wantedRt, err := stateutil.Uint64ListRootWithRegistryLimit(st.balances)
	assert.NoError(t, err)
	assert.Equal(t, wantedRt, newRt, "state roots are unequal")
}
