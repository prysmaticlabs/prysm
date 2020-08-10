package kv

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestVerifySlotsPerArchivePoint(t *testing.T) {
	db := setupDB(t)

	// This should set default to 2048.
	require.NoError(t, db.verifySlotsPerArchivePoint())

	// This should not fail with default 2048.
	require.NoError(t, db.verifySlotsPerArchivePoint())

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.SlotsPerArchivedPoint = 256
	params.OverrideBeaconConfig(config)

	// This should fail.
	msg := "could not update --slots-per-archive-point after it has been set"
	assert.ErrorContains(t, msg, db.verifySlotsPerArchivePoint())
}
