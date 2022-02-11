package validators

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestHasVoted_OK(t *testing.T) {
	// Setting bitlist to 11111111.
	pendingAttestation := &ethpb.Attestation{
		AggregationBits: []byte{0xFF, 0x01},
	}

	for i := uint64(0); i < pendingAttestation.AggregationBits.Len(); i++ {
		assert.Equal(t, true, pendingAttestation.AggregationBits.BitAt(i), "Validator voted but received didn't vote")
	}

	// Setting bit field to 10101010.
	pendingAttestation = &ethpb.Attestation{
		AggregationBits: []byte{0xAA, 0x1},
	}

	for i := uint64(0); i < pendingAttestation.AggregationBits.Len(); i++ {
		voted := pendingAttestation.AggregationBits.BitAt(i)
		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestInitiateValidatorExit_AlreadyExited(t *testing.T) {
	exitEpoch := types.Epoch(199)
	base := &ethpb.BeaconState{Validators: []*ethpb.Validator{{
		ExitEpoch: exitEpoch},
	}}
	state, err := v1.InitializeFromProto(base)
	require.NoError(t, err)
	newState, err := InitiateValidatorExit(context.Background(), state, 0)
	require.NoError(t, err)
	v, err := newState.ValidatorAtIndex(0)
	require.NoError(t, err)
	assert.Equal(t, exitEpoch, v.ExitEpoch, "Already exited")
}

func TestInitiateValidatorExit_ProperExit(t *testing.T) {
	exitedEpoch := types.Epoch(100)
	idx := types.ValidatorIndex(3)
	base := &ethpb.BeaconState{Validators: []*ethpb.Validator{
		{ExitEpoch: exitedEpoch},
		{ExitEpoch: exitedEpoch + 1},
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
	}}
	state, err := v1.InitializeFromProto(base)
	require.NoError(t, err)
	newState, err := InitiateValidatorExit(context.Background(), state, idx)
	require.NoError(t, err)
	v, err := newState.ValidatorAtIndex(idx)
	require.NoError(t, err)
	assert.Equal(t, exitedEpoch+2, v.ExitEpoch, "Exit epoch was not the highest")
}

func TestInitiateValidatorExit_ChurnOverflow(t *testing.T) {
	exitedEpoch := types.Epoch(100)
	idx := types.ValidatorIndex(4)
	base := &ethpb.BeaconState{Validators: []*ethpb.Validator{
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: exitedEpoch + 2}, // overflow here
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
	}}
	state, err := v1.InitializeFromProto(base)
	require.NoError(t, err)
	newState, err := InitiateValidatorExit(context.Background(), state, idx)
	require.NoError(t, err)

	// Because of exit queue overflow,
	// validator who init exited has to wait one more epoch.
	v, err := newState.ValidatorAtIndex(0)
	require.NoError(t, err)
	wantedEpoch := v.ExitEpoch + 1

	v, err = newState.ValidatorAtIndex(idx)
	require.NoError(t, err)
	assert.Equal(t, wantedEpoch, v.ExitEpoch, "Exit epoch did not cover overflow case")
}

func TestSlashValidator_OK(t *testing.T) {
	validatorCount := 100
	registry := make([]*ethpb.Validator, 0, validatorCount)
	balances := make([]uint64, 0, validatorCount)
	for i := 0; i < validatorCount; i++ {
		registry = append(registry, &ethpb.Validator{
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		})
		balances = append(balances, params.BeaconConfig().MaxEffectiveBalance)
	}

	base := &ethpb.BeaconState{
		Validators:  registry,
		Slashings:   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Balances:    balances,
	}
	state, err := v1.InitializeFromProto(base)
	require.NoError(t, err)

	slashedIdx := types.ValidatorIndex(2)

	proposer, err := helpers.BeaconProposerIndex(context.Background(), state)
	require.NoError(t, err, "Could not get proposer")
	proposerBal, err := state.BalanceAtIndex(proposer)
	require.NoError(t, err)
	cfg := params.BeaconConfig()
	slashedState, err := SlashValidator(context.Background(), state, slashedIdx, cfg.MinSlashingPenaltyQuotient, cfg.ProposerRewardQuotient)
	require.NoError(t, err, "Could not slash validator")
	require.Equal(t, true, slashedState.Version() == version.Phase0)

	v, err := state.ValidatorAtIndex(slashedIdx)
	require.NoError(t, err)
	assert.Equal(t, true, v.Slashed, "Validator not slashed despite supposed to being slashed")
	assert.Equal(t, time.CurrentEpoch(state)+params.BeaconConfig().EpochsPerSlashingsVector, v.WithdrawableEpoch, "Withdrawable epoch not the expected value")

	maxBalance := params.BeaconConfig().MaxEffectiveBalance
	slashedBalance := state.Slashings()[state.Slot().Mod(uint64(params.BeaconConfig().EpochsPerSlashingsVector))]
	assert.Equal(t, maxBalance, slashedBalance, "Slashed balance isnt the expected amount")

	whistleblowerReward := slashedBalance / params.BeaconConfig().WhistleBlowerRewardQuotient
	bal, err := state.BalanceAtIndex(proposer)
	require.NoError(t, err)
	// The proposer is the whistleblower in phase 0.
	assert.Equal(t, proposerBal+whistleblowerReward, bal, "Did not get expected balance for proposer")
	bal, err = state.BalanceAtIndex(slashedIdx)
	require.NoError(t, err)
	v, err = state.ValidatorAtIndex(slashedIdx)
	require.NoError(t, err)
	assert.Equal(t, maxBalance-(v.EffectiveBalance/params.BeaconConfig().MinSlashingPenaltyQuotient), bal, "Did not get expected balance for slashed validator")
}

func TestActivatedValidatorIndices(t *testing.T) {
	tests := []struct {
		state  *ethpb.BeaconState
		wanted []types.ValidatorIndex
	}{
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						ActivationEpoch: 0,
						ExitEpoch:       1,
					},
					{
						ActivationEpoch: 0,
						ExitEpoch:       1,
					},
					{
						ActivationEpoch: 5,
					},
					{
						ActivationEpoch: 0,
						ExitEpoch:       1,
					},
				},
			},
			wanted: []types.ValidatorIndex{0, 1, 3},
		},
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						ActivationEpoch: helpers.ActivationExitEpoch(10),
					},
				},
			},
			wanted: []types.ValidatorIndex{},
		},
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						ActivationEpoch: 0,
						ExitEpoch:       1,
					},
				},
			},
			wanted: []types.ValidatorIndex{0},
		},
	}
	for _, tt := range tests {
		s, err := v1.InitializeFromProto(tt.state)
		require.NoError(t, err)
		activatedIndices := ActivatedValidatorIndices(time.CurrentEpoch(s), tt.state.Validators)
		assert.DeepEqual(t, tt.wanted, activatedIndices)
	}
}

func TestSlashedValidatorIndices(t *testing.T) {
	tests := []struct {
		state  *ethpb.BeaconState
		wanted []types.ValidatorIndex
	}{
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector,
						Slashed:           true,
					},
					{
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector,
						Slashed:           false,
					},
					{
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector,
						Slashed:           true,
					},
				},
			},
			wanted: []types.ValidatorIndex{0, 2},
		},
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector,
					},
				},
			},
			wanted: []types.ValidatorIndex{},
		},
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector,
						Slashed:           true,
					},
				},
			},
			wanted: []types.ValidatorIndex{0},
		},
	}
	for _, tt := range tests {
		s, err := v1.InitializeFromProto(tt.state)
		require.NoError(t, err)
		slashedIndices := SlashedValidatorIndices(time.CurrentEpoch(s), tt.state.Validators)
		assert.DeepEqual(t, tt.wanted, slashedIndices)
	}
}

func TestExitedValidatorIndices(t *testing.T) {
	tests := []struct {
		state  *ethpb.BeaconState
		wanted []types.ValidatorIndex
	}{
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
						ExitEpoch:         0,
						WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
					},
					{
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
						ExitEpoch:         0,
						WithdrawableEpoch: 10,
					},
					{
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
						ExitEpoch:         0,
						WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
					},
				},
			},
			wanted: []types.ValidatorIndex{0, 2},
		},
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
						ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
						WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
					},
				},
			},
			wanted: []types.ValidatorIndex{},
		},
		{
			state: &ethpb.BeaconState{
				Validators: []*ethpb.Validator{
					{
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
						ExitEpoch:         0,
						WithdrawableEpoch: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
					},
				},
			},
			wanted: []types.ValidatorIndex{0},
		},
	}
	for _, tt := range tests {
		s, err := v1.InitializeFromProto(tt.state)
		require.NoError(t, err)
		activeCount, err := helpers.ActiveValidatorCount(context.Background(), s, time.PrevEpoch(s))
		require.NoError(t, err)
		exitedIndices, err := ExitedValidatorIndices(0, tt.state.Validators, activeCount)
		require.NoError(t, err)
		assert.DeepEqual(t, tt.wanted, exitedIndices)
	}
}
