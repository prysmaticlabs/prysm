package kv

import (
	"context"
	"fmt"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestStore_GenesisValidatorsRoot_ReadAndWrite(t *testing.T) {
	ctx := context.Background()

	subTests := []struct {
		name    string
		init    []byte
		want    []byte
		write   []byte
		wantErr bool
	}{
		{
			name:    "empty then write",
			init:    nil,
			want:    params.BeaconConfig().ZeroHash[:],
			write:   params.BeaconConfig().ZeroHash[:],
			wantErr: false,
		},
		{
			name:    "zero then overwrite with the same value",
			init:    params.BeaconConfig().ZeroHash[:],
			want:    params.BeaconConfig().ZeroHash[:],
			write:   params.BeaconConfig().ZeroHash[:],
			wantErr: false,
		},
		{
			name:    "zero then overwrite with a different value",
			init:    params.BeaconConfig().ZeroHash[:],
			want:    params.BeaconConfig().ZeroHash[:],
			write:   []byte{5},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		for _, st := range subTests {
			t.Run(fmt.Sprintf("%s - %s", tt.name, st.name), func(t *testing.T) {
				// Initialize the database with the initial value.
				db := setupDB(t, [][fieldparams.BLSPubkeyLength]byte{}, tt.slashingProtectionType)
				err := db.SaveGenesisValidatorsRoot(ctx, st.init)
				require.NoError(t, err)

				// Read the value from the database (just to ensure our setup is OK).
				got, err := db.GenesisValidatorsRoot(ctx)
				require.NoError(t, err)
				require.DeepEqual(t, st.init, got)

				// Write the value to the database.
				err = db.SaveGenesisValidatorsRoot(ctx, st.write)
				if (err != nil) != st.wantErr {
					t.Errorf("GenesisValidatorsRoot() error = %v, wantErr %v", err, st.wantErr)
				}

				// Read the value from the database.
				got, err = db.GenesisValidatorsRoot(ctx)
				require.NoError(t, err)
				require.DeepEqual(t, st.want, got)

				// Close the database.
				require.NoError(t, db.Close())
			})
		}
	}
}
