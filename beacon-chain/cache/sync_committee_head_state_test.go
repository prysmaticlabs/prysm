package cache

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncCommitteeHeadState(t *testing.T) {
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
				state: &v2.BeaconState{},
			},
			wantErr: true,
		},
		{
			name: "found with key",
			key:  types.Slot(1),
			put: &put{
				slot:  types.Slot(1),
				state: &v2.BeaconState{},
			},
			want: &v2.BeaconState{},
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
