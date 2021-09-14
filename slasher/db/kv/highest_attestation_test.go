package kv

import (
	"context"
	"testing"

	slashbp "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSaveHighestAttestation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		name         string
		toSave       []*slashbp.HighestAttestation
		cacheEnabled bool
	}{
		{
			name: "save to cache",
			toSave: []*slashbp.HighestAttestation{
				{
					HighestTargetEpoch: 1,
					HighestSourceEpoch: 0,
					ValidatorIndex:     1,
				},
			},
			cacheEnabled: true,
		},
		{
			name: "save to db",
			toSave: []*slashbp.HighestAttestation{
				{
					HighestTargetEpoch: 1,
					HighestSourceEpoch: 0,
					ValidatorIndex:     2,
				},
			},
			cacheEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, att := range tt.toSave {
				db.highestAttCacheEnabled = tt.cacheEnabled

				require.NoError(t, db.SaveHighestAttestation(ctx, att), "Save highest attestation failed")

				found, err := db.HighestAttestation(ctx, att.ValidatorIndex)
				require.NoError(t, err)
				require.NotNil(t, found)
				require.Equal(t, att.ValidatorIndex, found.ValidatorIndex)
				require.Equal(t, att.HighestSourceEpoch, found.HighestSourceEpoch)
				require.Equal(t, att.HighestTargetEpoch, found.HighestTargetEpoch)
			}
		})
	}
}

func TestFetchNonExistingHighestAttestation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	t.Run("cached", func(t *testing.T) {
		db.highestAttCacheEnabled = true
		found, err := db.HighestAttestation(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, (*slashbp.HighestAttestation)(nil), found, "should not find HighestAttestation")
	})

	t.Run("disk", func(t *testing.T) {
		db.highestAttCacheEnabled = false
		found, err := db.HighestAttestation(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, (*slashbp.HighestAttestation)(nil), found, "should not find HighestAttestation")
	})

}
