package kv

import (
	"context"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStateSummary_CanSaveRretrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	r1 := bytesutil.ToBytes32([]byte{'A'})
	r2 := bytesutil.ToBytes32([]byte{'B'})
	s1 := &pb.StateSummary{Slot: 1, Root: r1[:]}

	// State summary should not exist yet.
	require.Equal(t, false, db.HasStateSummary(ctx, r1), "State summary should not be saved")
	require.NoError(t, db.SaveStateSummary(ctx, s1))
	require.Equal(t, true, db.HasStateSummary(ctx, r1), "State summary should be saved")

	saved, err := db.StateSummary(ctx, r1)
	require.NoError(t, err)
	assert.DeepEqual(t, s1, saved, "State summary does not equal")

	// Save a new state summary.
	s2 := &pb.StateSummary{Slot: 2, Root: r2[:]}

	// State summary should not exist yet.
	require.Equal(t, false, db.HasStateSummary(ctx, r2), "State summary should not be saved")
	require.NoError(t, db.SaveStateSummary(ctx, s2))
	require.Equal(t, true, db.HasStateSummary(ctx, r2), "State summary should be saved")

	saved, err = db.StateSummary(ctx, r2)
	require.NoError(t, err)
	assert.DeepEqual(t, s2, saved, "State summary does not equal")
}
