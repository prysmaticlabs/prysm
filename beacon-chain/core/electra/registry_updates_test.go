package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestProcessRegistryUpdates(t *testing.T) {
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
					require.Equal(t, params.BeaconConfig().MaxSeedLookahead, val.ExitEpoch, "Could not update registry %d", i)
				}
			},
		},
		{
			name: "Eligible to activate", // This test case is written for phase0, now failing in electra. TODO: Update test case.
			state: func() state.BeaconState {
				base := &eth.BeaconState{
					Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
					FinalizedCheckpoint: &eth.Checkpoint{Epoch: 6, Root: make([]byte, fieldparams.RootLength)},
				}
				limit := helpers.ValidatorActivationChurnLimit(0)
				for i := uint64(0); i < limit+10; i++ {
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
				currentEpoch := time.CurrentEpoch(st)
				limit := helpers.ValidatorActivationChurnLimit(0)

				for i, val := range st.Validators() {
					require.Equal(t, currentEpoch+1, val.ActivationEligibilityEpoch, "Could not update registry %d, unexpected activation eligibility epoch", i)
					if uint64(i) < limit && val.ActivationEpoch != helpers.ActivationExitEpoch(currentEpoch) {
						t.Errorf("Could not update registry %d, validators failed to activate: wanted activation epoch %d, got %d",
							i, helpers.ActivationExitEpoch(currentEpoch), val.ActivationEpoch)
					}
					if uint64(i) >= limit && val.ActivationEpoch != params.BeaconConfig().FarFutureEpoch {
						t.Errorf("Could not update registry %d, validators should not have been activated, wanted activation epoch: %d, got %d",
							i, params.BeaconConfig().FarFutureEpoch, val.ActivationEpoch)
					}
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
