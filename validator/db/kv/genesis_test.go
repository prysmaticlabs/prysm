package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_GenesisValidatorRoot_ReadAndWrite(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t, [][48]byte{})
	tests := []struct {
		name    string
		want    []byte
		write   []byte
		wantErr bool
	}{
		{
			name:  "empty then write",
			want:  []byte{},
			write: []byte{1},
		},
		{
			name:  "1 then overwrite",
			want:  []byte{1},
			write: []byte{5},
		},
		{
			name:  "5 then zerohash",
			want:  []byte{1},
			write: params.BeaconConfig().ZeroHash[:],
		},
		{
			name:  "zerohash",
			want:  params.BeaconConfig().ZeroHash[:],
			write: params.BeaconConfig().ZeroHash[:],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GenesisValidatorRoot(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisValidatorRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, got, tt.want)
			require.NoError(t, db.SaveGenesisValidatorRoot(ctx, tt.write))
		})
	}
}
