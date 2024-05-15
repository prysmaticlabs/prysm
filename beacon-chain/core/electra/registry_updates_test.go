package electra_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
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
			name: "No rotation",
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
