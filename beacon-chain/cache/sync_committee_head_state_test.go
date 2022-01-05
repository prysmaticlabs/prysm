package cache

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state-native/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state-native/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state-native/v3"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSyncCommitteeHeadState(t *testing.T) {
	beaconState, err := v2.InitializeFromProto(&ethpb.BeaconStateAltair{
		Fork: &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	require.NoError(t, err)
	phase0State, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Fork: &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	require.NoError(t, err)
	mergeState, err := v3.InitializeFromProto(&ethpb.BeaconStateMerge{
		Fork: &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	require.NoError(t, err)
	type put struct {
		slot  types.Slot
		state state.BeaconState
	}
	tests := []struct {
		name       string
		key        types.Slot
		put        *put
		want       state.BeaconState
		wantErr    bool
		wantPutErr bool
	}{
		{
			name: "putting error in",
			key:  types.Slot(1),
			put: &put{
				slot:  types.Slot(1),
				state: nil,
			},
			wantPutErr: true,
			wantErr:    true,
		},
		{
			name: "putting invalid state in",
			key:  types.Slot(1),
			put: &put{
				slot:  types.Slot(1),
				state: phase0State,
			},
			wantPutErr: true,
			wantErr:    true,
		},
		{
			name:    "not found when empty cache",
			key:     types.Slot(1),
			wantErr: true,
		},
		{
			name: "not found when non-existent key in non-empty cache",
			key:  types.Slot(2),
			put: &put{
				slot:  types.Slot(1),
				state: beaconState,
			},
			wantErr: true,
		},
		{
			name: "found with key",
			key:  types.Slot(1),
			put: &put{
				slot:  types.Slot(1),
				state: beaconState,
			},
			want: beaconState,
		},
		{
			name: "not found when non-existent key in non-empty cache (merge state)",
			key:  types.Slot(2),
			put: &put{
				slot:  types.Slot(1),
				state: mergeState,
			},
			wantErr: true,
		},
		{
			name: "found with key (merge state)",
			key:  types.Slot(100),
			put: &put{
				slot:  types.Slot(100),
				state: mergeState,
			},
			want: mergeState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSyncCommitteeHeadState()
			if tt.put != nil {
				err := c.Put(tt.put.slot, tt.put.state)
				if (err != nil) != tt.wantPutErr {
					t.Fatalf("Put() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
			got, err := c.Get(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			require.DeepEqual(t, tt.want, got)
		})
	}
}
