package kv

import (
	"context"
	"flag"
	ethereum_slashing "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/urfave/cli/v2"
	"testing"
)

func TestSaveHighestAttestation(t *testing.T) {
	app := &cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	ctx := context.Background()

	tests := []struct{
		name string
		toSave []*ethereum_slashing.HighestAttestation
		cacheEnabled bool
	}{
		{
			name:"save to cache",
			toSave: []*ethereum_slashing.HighestAttestation {
				&ethereum_slashing.HighestAttestation{
					HighestTargetEpoch:1,
					HighestSourceEpoch:0,
					ValidatorId: 1,
				},
			},
			cacheEnabled: true,
		},
		{
			name:"save to db",
			toSave: []*ethereum_slashing.HighestAttestation {
				&ethereum_slashing.HighestAttestation{
					HighestTargetEpoch:1,
					HighestSourceEpoch:0,
					ValidatorId: 2,
				},
			},
			cacheEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func (t *testing.T) {
			for _, att := range tt.toSave {
				db.highestAttCacheEnabled = tt.cacheEnabled

				require.NoError(t, db.SaveHighestAttestation(ctx, att), "Save highest attestation failed")

				found, err := db.HighestAttestation(ctx, att.ValidatorId)
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, att.ValidatorId, found.ValidatorId)
				require.Equal(t, att.HighestSourceEpoch, found.HighestSourceEpoch)
				require.Equal(t, att.HighestTargetEpoch, found.HighestTargetEpoch)
			}
		})
	}
}
