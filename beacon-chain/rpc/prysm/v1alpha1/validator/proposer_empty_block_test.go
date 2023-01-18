package validator

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func Test_getEmptyBlock(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.AltairForkEpoch = 1
	config.BellatrixForkEpoch = 2
	config.CapellaForkEpoch = 3
	params.OverrideBeaconConfig(config)

	tests := []struct {
		name string
		slot types.Slot
		want func() interfaces.SignedBeaconBlock
	}{
		{
			name: "altair",
			slot: types.Slot(params.BeaconConfig().AltairForkEpoch) * params.BeaconConfig().SlotsPerEpoch,
			want: func() interfaces.SignedBeaconBlock {
				b, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}}})
				require.NoError(t, err)
				return b
			},
		},
		{
			name: "bellatrix",
			slot: types.Slot(params.BeaconConfig().BellatrixForkEpoch) * params.BeaconConfig().SlotsPerEpoch,
			want: func() interfaces.SignedBeaconBlock {
				b, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{Body: &ethpb.BeaconBlockBodyBellatrix{}}})
				require.NoError(t, err)
				return b
			},
		},
		{
			name: "capella",
			slot: types.Slot(params.BeaconConfig().CapellaForkEpoch) * params.BeaconConfig().SlotsPerEpoch,
			want: func() interfaces.SignedBeaconBlock {
				b, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockCapella{Block: &ethpb.BeaconBlockCapella{Body: &ethpb.BeaconBlockBodyCapella{}}})
				require.NoError(t, err)
				return b
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getEmptyBlock(tt.slot)
			require.NoError(t, err)
			require.DeepEqual(t, tt.want(), got, "getEmptyBlock() = %v, want %v", got, tt.want())
		})
	}
}
