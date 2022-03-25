package wrapper_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestWrappedSignedBeaconBlock(t *testing.T) {
	tests := []struct {
		name    string
		blk     interface{}
		wantErr bool
	}{
		{
			name:    "unsupported type",
			blk:     "not a beacon block",
			wantErr: true,
		},
		{
			name: "phase0",
			blk:  util.NewBeaconBlock(),
		},
		{
			name: "altair",
			blk:  util.NewBeaconBlockAltair(),
		},
		{
			name: "bellatrix",
			blk:  util.NewBeaconBlockBellatrix(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := wrapper.WrappedSignedBeaconBlock(tt.blk)
			if tt.wantErr {
				require.ErrorIs(t, err, wrapper.ErrUnsupportedSignedBeaconBlock)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
