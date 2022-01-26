package kv

import (
	"context"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestStore_GenesisValidatorsRoot_ReadAndWrite(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t, [][fieldparams.BLSPubkeyLength]byte{})
	tests := []struct {
		name    string
		want    []byte
		write   []byte
		wantErr bool
	}{
		{
			name:  "empty then write",
			want:  nil,
			write: params.BeaconConfig().ZeroHash[:],
		},
		{
			name:    "zero then overwrite rejected",
			want:    params.BeaconConfig().ZeroHash[:],
			write:   []byte{5},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GenesisValidatorsRoot(ctx)
			require.NoError(t, err)
			require.DeepEqual(t, tt.want, got)
			err = db.SaveGenesisValidatorsRoot(ctx, tt.write)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisValidatorRoot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
