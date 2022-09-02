package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestStateSummary_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	r1, err := util.SaveBlock(t, ctx, db, b1).Block().HashTreeRoot()
	require.NoError(t, err)
	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 2
	r2, err := util.SaveBlock(t, ctx, db, b2).Block().HashTreeRoot()
	require.NoError(t, err)
	s1 := &ethpb.StateSummary{Slot: 1, Root: r1[:]}

	// State summary should not exist yet.
	require.Equal(t, false, db.HasStateSummary(ctx, r1), "State summary should not be saved")
	require.NoError(t, db.SaveStateSummary(ctx, s1))
	require.Equal(t, true, db.HasStateSummary(ctx, r1), "State summary should be saved")

	saved, err := db.StateSummary(ctx, r1)
	require.NoError(t, err)
	assert.DeepEqual(t, s1, saved, "State summary does not equal")

	// Save a new state summary.
	s2 := &ethpb.StateSummary{Slot: 2, Root: r2[:]}

	// State summary should not exist yet.
	require.Equal(t, false, db.HasStateSummary(ctx, r2), "State summary should not be saved")
	require.NoError(t, db.SaveStateSummary(ctx, s2))
	require.Equal(t, true, db.HasStateSummary(ctx, r2), "State summary should be saved")

	saved, err = db.StateSummary(ctx, r2)
	require.NoError(t, err)
	assert.DeepEqual(t, s2, saved, "State summary does not equal")
}

func TestStateSummary_CacheToDB(t *testing.T) {
	db := setupDB(t)

	ctx := context.Background()

	summaries := make([]*ethpb.StateSummary, stateSummaryCachePruneCount-1)
	roots := make([][32]byte, stateSummaryCachePruneCount-1)
	for i := range summaries {
		b := util.NewBeaconBlock()
		b.Block.Slot = types.Slot(i)
		b.Block.Body.Graffiti = bytesutil.PadTo([]byte{byte(i)}, 32)
		r, err := util.SaveBlock(t, ctx, db, b).Block().HashTreeRoot()
		require.NoError(t, err)
		summaries[i] = &ethpb.StateSummary{Slot: types.Slot(i), Root: r[:]}
		roots[i] = r
	}

	require.NoError(t, db.SaveStateSummaries(context.Background(), summaries))
	require.Equal(t, db.stateSummaryCache.len(), stateSummaryCachePruneCount-1)

	b := util.NewBeaconBlock()
	b.Block.Slot = types.Slot(1000)
	r, err := util.SaveBlock(t, ctx, db, b).Block().HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, db.SaveStateSummary(context.Background(), &ethpb.StateSummary{Slot: 1000, Root: r[:]}))
	require.Equal(t, db.stateSummaryCache.len(), stateSummaryCachePruneCount)

	b = util.NewBeaconBlock()
	b.Block.Slot = 1001

	r, err = util.SaveBlock(t, ctx, db, b).Block().HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, db.SaveStateSummary(context.Background(), &ethpb.StateSummary{Slot: 1001, Root: r[:]}))
	require.Equal(t, db.stateSummaryCache.len(), 1)

	for _, r := range roots {
		require.Equal(t, true, db.HasStateSummary(context.Background(), r))
	}
}

func TestStateSummary_CacheToDB_FailsIfMissingBlock(t *testing.T) {
	db := setupDB(t)

	ctx := context.Background()

	summaries := make([]*ethpb.StateSummary, stateSummaryCachePruneCount-1)
	for i := range summaries {
		b := util.NewBeaconBlock()
		b.Block.Slot = types.Slot(i)
		b.Block.Body.Graffiti = bytesutil.PadTo([]byte{byte(i)}, 32)
		r, err := util.SaveBlock(t, ctx, db, b).Block().HashTreeRoot()
		require.NoError(t, err)
		summaries[i] = &ethpb.StateSummary{Slot: types.Slot(i), Root: r[:]}
	}

	require.NoError(t, db.SaveStateSummaries(context.Background(), summaries))
	require.Equal(t, db.stateSummaryCache.len(), stateSummaryCachePruneCount-1)

	junkRoot := [32]byte{1, 2, 3}

	require.NoError(t, db.SaveStateSummary(context.Background(), &ethpb.StateSummary{Slot: 1000, Root: junkRoot[:]}))
	require.Equal(t, db.stateSummaryCache.len(), stateSummaryCachePruneCount)

	// Next insertion causes the buffer to flush.
	b := util.NewBeaconBlock()
	b.Block.Slot = 1001
	r, err := util.SaveBlock(t, ctx, db, b).Block().HashTreeRoot()
	require.NoError(t, err)
	require.ErrorIs(t, db.SaveStateSummary(context.Background(), &ethpb.StateSummary{Slot: 1001, Root: r[:]}), ErrNotFoundBlock)

	require.NoError(t, db.deleteStateSummary(junkRoot)) // Delete bad summary or db will throw an error on test cleanup.
}

func TestStateSummary_CanDelete(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	r1 := bytesutil.ToBytes32([]byte{'A'})
	s1 := &ethpb.StateSummary{Slot: 1, Root: r1[:]}

	require.Equal(t, false, db.HasStateSummary(ctx, r1), "State summary should not be saved")
	require.NoError(t, db.SaveStateSummary(ctx, s1))
	require.Equal(t, true, db.HasStateSummary(ctx, r1), "State summary should be saved")

	require.NoError(t, db.deleteStateSummary(r1))
	require.Equal(t, false, db.HasStateSummary(ctx, r1), "State summary should not be saved")
}
