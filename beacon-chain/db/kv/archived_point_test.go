package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestArchivedPointIndexRoot_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	i1 := types.Slot(100)
	r1 := [32]byte{'A'}

	received := db.ArchivedPointRoot(ctx, i1)
	require.NotEqual(t, r1, received, "Should not have been saved")
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, st.SetSlot(i1))
	require.NoError(t, db.SaveState(ctx, st, r1))
	received = db.ArchivedPointRoot(ctx, i1)
	assert.Equal(t, r1, received, "Should have been saved")
}
