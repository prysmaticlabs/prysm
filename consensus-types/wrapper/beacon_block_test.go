package wrapper

import (
	"testing"

	typeerrors "github.com/prysmaticlabs/prysm/consensus-types/errors"
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
			_, err := WrappedSignedBeaconBlock(tt.blk)
			if tt.wantErr {
				require.ErrorIs(t, err, typeerrors.ErrUnsupportedSignedBeaconBlock)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
