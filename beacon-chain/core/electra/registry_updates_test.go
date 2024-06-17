package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestProcessRegistryUpdates(t *testing.T) {
	const electraEpoch = 3
	cfg := params.BeaconConfig()
	cfg.ElectraForkEpoch = electraEpoch
	params.SetActiveTestCleanup(t, cfg)

	tests := []struct {
		name  string
		state state.BeaconState
		check func(*testing.T, state.BeaconState)
	}{
		{
			name: "No rotation", // No validators exited.
			state: func() state.BeaconState {
				base := &eth.BeaconState{
					Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
					Validators: []*eth.Validator{
						{ExitEpoch: params.BeaconConfig().MaxSeedLookahead},
						{ExitEpoch: params.BeaconConfig().MaxSeedLookahead},
					},
					Balances: []uint64{
						params.BeaconConfig().MaxEffectiveBalance,
						params.BeaconConfig().MaxEffectiveBalance,
					},
					FinalizedCheckpoint: &eth.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				}
				st, err := state_native.InitializeFromProtoPhase0(base)
				require.NoError(t, err)
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				for i, val := range st.Validators() {
					require.Equal(t, params.BeaconConfig().MaxSeedLookahead, val.ExitEpoch, "validator updated unexpectedly at index %d", i)
				}
			},
		},
		{
			name: "Validators are activated",
			state: func() state.BeaconState {
				// Construct a state which
				base := &eth.BeaconState{
					Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
					FinalizedCheckpoint: &eth.Checkpoint{Epoch: 6, Root: make([]byte, fieldparams.RootLength)},
				}
				for i := uint64(0); i < 10; i++ {
					base.Validators = append(base.Validators, &eth.Validator{
						ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
						EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
						ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
					})
				}
				st, err := state_native.InitializeFromProtoPhase0(base)
				require.NoError(t, err)
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				activationEpoch := helpers.ActivationExitEpoch(5)
				// All validators should be activated.
				for i, val := range st.Validators() {
					require.Equal(t, activationEpoch, val.ActivationEpoch, "failed to update validator at index %d", i)
				}
			},
		},
		{
			name: "Validators are exited",
			state: func() state.BeaconState {
				// Construct a state which
				base := &eth.BeaconState{
					Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
					FinalizedCheckpoint: &eth.Checkpoint{Epoch: 6, Root: make([]byte, fieldparams.RootLength)},
				}
				for i := uint64(0); i < 10; i++ {
					base.Validators = append(base.Validators, &eth.Validator{
						EffectiveBalance:  params.BeaconConfig().EjectionBalance - 1,
						ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
						WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
					})
				}
				st, err := state_native.InitializeFromProtoPhase0(base)
				require.NoError(t, err)
				return st
			}(),
			check: func(t *testing.T, st state.BeaconState) {
				// All validators should be exited
				for i, val := range st.Validators() {
					require.NotEqual(t, params.BeaconConfig().FarFutureEpoch, val.ExitEpoch, "failed to update exit epoch on validator %d", i)
					require.NotEqual(t, params.BeaconConfig().FarFutureEpoch, val.WithdrawableEpoch, "failed to update withdrawable epoch on validator %d", i)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := electra.ProcessRegistryUpdates(context.TODO(), tt.state)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, res)
			}
		})
	}
}
