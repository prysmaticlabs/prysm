package altair_test

import (
	"bytes"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	testAltair "github.com/prysmaticlabs/prysm/shared/testutil/altair"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncCommitteeIndices_CanGet(t *testing.T) {
	getState := func(t *testing.T, count uint64) *stateAltair.BeaconState {
		validators := make([]*ethpb.Validator, count)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().MinDepositAmount,
			}
		}
		state, err := stateAltair.InitializeFromProto(&pb.BeaconStateAltair{
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		require.NoError(t, err)
		return state
	}

	type args struct {
		state *stateAltair.BeaconState
		epoch types.Epoch
	}
	tests := []struct {
		name      string
		args      args
		wantErr   bool
		errString string
	}{
		{
			name: "nil state",
			args: args{
				state: nil,
			},
			wantErr:   true,
			errString: "nil inner state",
		},
		{
			name: "genesis validator count, epoch 0",
			args: args{
				state: getState(t, params.BeaconConfig().MinGenesisActiveValidatorCount),
				epoch: 0,
			},
			wantErr: false,
		},
		{
			name: "genesis validator count, epoch 100",
			args: args{
				state: getState(t, params.BeaconConfig().MinGenesisActiveValidatorCount),
				epoch: 100,
			},
			wantErr: false,
		},
		{
			name: "less than optimal validator count, epoch 100",
			args: args{
				state: getState(t, params.BeaconConfig().MaxValidatorsPerCommittee),
				epoch: 100,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()
			got, err := NextSyncCommitteeIndices(tt.args.state)
			if tt.wantErr {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, int(params.BeaconConfig().SyncCommitteeSize), len(got))
			}
		})
	}
}

func TestSyncCommitteeIndices_DifferentPeriods(t *testing.T) {
	helpers.ClearCache()
	getState := func(t *testing.T, count uint64) *stateAltair.BeaconState {
		validators := make([]*ethpb.Validator, count)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().MinDepositAmount,
			}
		}
		state, err := stateAltair.InitializeFromProto(&pb.BeaconStateAltair{
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		require.NoError(t, err)
		return state
	}

	state := getState(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	got1, err := NextSyncCommitteeIndices(state)
	require.NoError(t, err)
	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	got2, err := NextSyncCommitteeIndices(state)
	require.NoError(t, err)
	require.DeepEqual(t, got1, got2)
	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch*types.Slot(params.BeaconConfig().EpochsPerSyncCommitteePeriod)))
	got2, err = NextSyncCommitteeIndices(state)
	require.NoError(t, err)
	require.DeepEqual(t, got1, got2)
	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch*types.Slot(2*params.BeaconConfig().EpochsPerSyncCommitteePeriod)))
	got2, err = NextSyncCommitteeIndices(state)
	require.NoError(t, err)
	require.DeepNotEqual(t, got1, got2)
}

func TestSyncCommittee_CanGet(t *testing.T) {
	getState := func(t *testing.T, count uint64) *stateAltair.BeaconState {
		validators := make([]*ethpb.Validator, count)
		for i := 0; i < len(validators); i++ {
			blsKey, err := bls.RandKey()
			require.NoError(t, err)
			validators[i] = &ethpb.Validator{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().MinDepositAmount,
				PublicKey:        blsKey.PublicKey().Marshal(),
			}
		}
		state, err := stateAltair.InitializeFromProto(&pb.BeaconStateAltair{
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		require.NoError(t, err)
		return state
	}

	type args struct {
		state *stateAltair.BeaconState
		epoch types.Epoch
	}
	tests := []struct {
		name      string
		args      args
		wantErr   bool
		errString string
	}{
		{
			name: "nil state",
			args: args{
				state: nil,
			},
			wantErr:   true,
			errString: "nil inner state",
		},
		{
			name: "genesis validator count, epoch 0",
			args: args{
				state: getState(t, params.BeaconConfig().MinGenesisActiveValidatorCount),
				epoch: 0,
			},
			wantErr: false,
		},
		{
			name: "genesis validator count, epoch 100",
			args: args{
				state: getState(t, params.BeaconConfig().MinGenesisActiveValidatorCount),
				epoch: 100,
			},
			wantErr: false,
		},
		{
			name: "less than optimal validator count, epoch 100",
			args: args{
				state: getState(t, params.BeaconConfig().MaxValidatorsPerCommittee),
				epoch: 100,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.ClearCache()
			require.NoError(t, tt.args.state.SetSlot(types.Slot(tt.args.epoch)*params.BeaconConfig().SlotsPerEpoch))
			got, err := NextSyncCommittee(tt.args.state)
			if tt.wantErr {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, int(params.BeaconConfig().SyncCommitteeSize), len(got.Pubkeys))
				require.Equal(t, params.BeaconConfig().BLSPubkeyLength, len(got.AggregatePubkey))
			}
		})
	}
}

func TestAssignedToSyncCommittee(t *testing.T) {
	s, _ := testAltair.DeterministicGenesisStateAltair(t, 5*params.BeaconConfig().SyncCommitteeSize)
	syncCommittee, err := altair.SyncCommittee(s, helpers.CurrentEpoch(s))
	require.NoError(t, err)
	require.NoError(t, s.SetCurrentSyncCommittee(syncCommittee))
	syncCommittee, err = altair.SyncCommittee(s, helpers.CurrentEpoch(s)+2*params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	require.NoError(t, err)
	require.NoError(t, s.SetNextSyncCommittee(syncCommittee))

	csc, err := s.CurrentSyncCommittee()
	require.NoError(t, err)
	nsc, err := s.NextSyncCommittee()
	require.NoError(t, err)

	currentSyncCommitteeIndex := 0
	vals := s.Validators()
	for i, val := range vals {
		if bytes.Equal(val.PublicKey, csc.Pubkeys[0]) {
			currentSyncCommitteeIndex = i
		}
	}
	nextSyncCommitteeIndex := 0
	for i, val := range vals {
		if bytes.Equal(val.PublicKey, nsc.Pubkeys[0]) {
			nextSyncCommitteeIndex = i
		}
	}

	tests := []struct {
		name   string
		epoch  types.Epoch
		check  types.ValidatorIndex
		exists bool
	}{
		{
			name:   "does not exist while asking current sync committee",
			epoch:  0,
			check:  0,
			exists: false,
		},
		{
			name:   "exists in current sync committee",
			epoch:  0,
			check:  types.ValidatorIndex(currentSyncCommitteeIndex),
			exists: true,
		},
		{
			name:   "exists in next sync committee",
			epoch:  256,
			check:  types.ValidatorIndex(nextSyncCommitteeIndex),
			exists: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := altair.AssignedToSyncCommittee(s, tt.epoch, tt.check)
			require.NoError(t, err)
			require.Equal(t, tt.exists, exists)
		})
	}
}

func TestAssignedToSyncCommittee_IncorrectEpoch(t *testing.T) {
	s, _ := testAltair.DeterministicGenesisStateAltair(t, 64)
	_, err := altair.AssignedToSyncCommittee(s, params.BeaconConfig().EpochsPerSyncCommitteePeriod*2, 0)
	require.ErrorContains(t, "epoch period 2 is not current period 0 or next period 1 in state", err)
}

func TestSubnetsForSyncCommittee(t *testing.T) {
	s, _ := testAltair.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	syncCommittee, err := altair.SyncCommittee(s, helpers.CurrentEpoch(s))
	require.NoError(t, err)
	require.NoError(t, s.SetCurrentSyncCommittee(syncCommittee))
	syncCommittee, err = altair.SyncCommittee(s, helpers.CurrentEpoch(s)+2*params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	require.NoError(t, err)
	require.NoError(t, s.SetNextSyncCommittee(syncCommittee))

	positions, err := altair.SubnetsForSyncCommittee(s, 0)
	require.NoError(t, err)
	require.DeepEqual(t, []uint64{4}, positions)
	positions, err = altair.SubnetsForSyncCommittee(s, 1)
	require.NoError(t, err)
	require.DeepEqual(t, []uint64{2}, positions)
	positions, err = altair.SubnetsForSyncCommittee(s, 2)
	require.NoError(t, err)
	require.DeepEqual(t, []uint64{7}, positions)
}

func TestSyncCommitteePeriod(t *testing.T) {
	tests := []struct {
		epoch  types.Epoch
		wanted uint64
	}{
		{epoch: 0, wanted: 0},
		{epoch: 0, wanted: 0 / uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)},
		{epoch: 1, wanted: 1 / uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)},
		{epoch: 1000, wanted: 1000 / uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)},
	}
	for _, test := range tests {
		require.Equal(t, test.wanted, altair.SyncCommitteePeriod(test.epoch))
	}
}
