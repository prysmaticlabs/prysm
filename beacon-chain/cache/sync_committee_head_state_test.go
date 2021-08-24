package cache

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncCommitteeHeadState(t *testing.T) {
	beaconState, err := v2.InitializeFromProto(&ethpb.BeaconStateAltair{
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
		name    string
		key     types.Slot
		put     *put
		want    state.BeaconState
		wantErr bool
	}{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSyncCommitteeHeadState()
			if tt.put != nil {
				c.Put(tt.put.slot, tt.put.state)
			}
			got, err := c.Get(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			require.DeepEqual(t, tt.want, got)
		})
	}
}
