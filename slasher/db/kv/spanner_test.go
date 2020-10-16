package kv

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

type spanMapTestStruct struct {
	epoch   uint64
	spanMap map[uint64]types.Span
}

var spanTests []spanMapTestStruct

func init() {
	spanTests = []spanMapTestStruct{
		{
			epoch: 1,
			spanMap: map[uint64]types.Span{
				1: {MinSpan: 10, MaxSpan: 20, HasAttested: false, SigBytes: [2]byte{1, 1}},
				2: {MinSpan: 11, MaxSpan: 21, HasAttested: true, SigBytes: [2]byte{1, 1}},
				3: {MinSpan: 12, MaxSpan: 22, HasAttested: false, SigBytes: [2]byte{1, 1}},
			},
		},
		{
			epoch: 2,
			spanMap: map[uint64]types.Span{
				1: {MinSpan: 10, MaxSpan: 20, HasAttested: false, SigBytes: [2]byte{1, 1}},
				2: {MinSpan: 11, MaxSpan: 21, HasAttested: true, SigBytes: [2]byte{1, 1}},
				3: {MinSpan: 12, MaxSpan: 22, HasAttested: true, SigBytes: [2]byte{1, 1}},
			},
		},
		{
			epoch: 3,
			spanMap: map[uint64]types.Span{
				1: {MinSpan: 10, MaxSpan: 20, HasAttested: true, SigBytes: [2]byte{1, 1}},
				2: {MinSpan: 11, MaxSpan: 21, SigBytes: [2]byte{1, 1}},
				3: {MinSpan: 12, MaxSpan: 22, SigBytes: [2]byte{1, 1}},
			},
		},
	}
}

func TestValidatorSpanMap_NilDB(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	validatorIdx := uint64(1)
	vsm, _, err := db.EpochSpansMap(ctx, validatorIdx)
	require.NoError(t, err, "Nil EpochSpansMap should not return error")
	require.DeepEqual(t, map[uint64]types.Span{}, vsm, "EpochSpansMap should return empty map")
}

func TestStore_SaveSpans(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveEpochSpansMap(ctx, tt.epoch, tt.spanMap)
		require.NoError(t, err, "Save validator span map failed")
		sm, _, err := db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err, "Failed to get validator span map")
		require.NotNil(t, sm)
		require.DeepEqual(t, tt.spanMap, sm, "Get should return validator span map")
		s, err := db.EpochSpanByValidatorIndex(ctx, 1, tt.epoch)
		require.NoError(t, err, "Failed to get validator span for epoch 1")
		require.DeepEqual(t, tt.spanMap[1], s, "Get should return validator spans for epoch 1")
	}
}

func TestStore_SaveCachedSpans(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveEpochSpansMap(ctx, tt.epoch, tt.spanMap)
		require.NoError(t, err, "Save validator span map failed")
		// wait for value to pass through cache buffers
		time.Sleep(time.Millisecond * 10)
		sm, _, err := db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err, "Failed to get validator span map")
		require.NotNil(t, sm)
		require.DeepEqual(t, tt.spanMap, sm, "Get should return validator span map")

		s, err := db.EpochSpanByValidatorIndex(ctx, 1, tt.epoch)
		require.NoError(t, err, "Failed to get validator span for epoch 1")
		require.DeepEqual(t, tt.spanMap[1], s, "Get should return validator spans for epoch 1")
	}
}

func TestStore_DeleteEpochSpans(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()
	db.spanCacheEnabled = false
	for _, tt := range spanTests {
		err := db.SaveEpochSpansMap(ctx, tt.epoch, tt.spanMap)
		require.NoError(t, err, "Save validator span map failed")
	}

	for _, tt := range spanTests {
		sm, _, err := db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err, "Failed to get validator span map")
		require.NotNil(t, sm)
		require.DeepEqual(t, tt.spanMap, sm, "Get should return validator span map")
		err = db.DeleteEpochSpans(ctx, tt.epoch)
		require.NoError(t, err, "Delete validator span map error")
		sm, _, err = db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err)
		require.DeepEqual(t, map[uint64]types.Span{}, sm, "Expected validator span map to be deleted")
	}
}

func TestValidatorSpanMap_DeletesOnCacheSavesToDB(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveEpochSpansMap(ctx, tt.epoch, tt.spanMap)
		require.NoError(t, err, "Save validator span map failed")
	}
	// Wait for value to pass through cache buffers.
	time.Sleep(time.Millisecond * 10)
	for _, tt := range spanTests {
		spanMap, _, err := db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err, "Failed to get validator span map")
		require.NotNil(t, spanMap)
		require.DeepEqual(t, tt.spanMap, spanMap, "Get should return validator span map")

		require.NoError(t, db.DeleteEpochSpans(ctx, tt.epoch), "Delete validator span map error")
		// Wait for value to pass through cache buffers.
		db.EnableSpanCache(false)
		time.Sleep(time.Millisecond * 10)
		spanMap, _, err = db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err)
		db.EnableSpanCache(true)
		require.DeepEqual(t, tt.spanMap, spanMap, "Expected validator span map to be deleted")
	}
}

func TestValidatorSpanMap_SaveOnEvict(t *testing.T) {
	db := setupDBDiffCacheSize(t, 5)
	ctx := context.Background()

	tsm := &spanMapTestStruct{
		epoch: 1,
		spanMap: map[uint64]types.Span{
			1: {MinSpan: 10, MaxSpan: 20, SigBytes: [2]byte{0, 1}},
			2: {MinSpan: 11, MaxSpan: 21, HasAttested: true},
			3: {MinSpan: 12, MaxSpan: 22},
		},
	}
	for i := uint64(0); i < 6; i++ {
		err := db.SaveEpochSpansMap(ctx, i, tsm.spanMap)
		require.NoError(t, err, "Save validator span map failed")
	}

	// Wait for value to pass through cache buffers.
	time.Sleep(time.Millisecond * 1000)
	for i := uint64(0); i < 6; i++ {
		sm, _, err := db.EpochSpansMap(ctx, i)
		require.NoError(t, err, "Failed to get validator span map")
		require.NotNil(t, sm)
		require.DeepEqual(t, tsm.spanMap, sm, "Get should return validator")
	}
}

func TestValidatorSpanMap_SaveCachedSpansMaps(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveEpochSpansMap(ctx, tt.epoch, tt.spanMap)
		require.NoError(t, err, "Save validator span map failed")
	}
	// wait for value to pass through cache buffers
	time.Sleep(time.Millisecond * 10)
	require.NoError(t, db.SaveCachedSpansMaps(ctx), "Failed to save cached span maps to db")
	db.spanCache.Purge()
	for _, tt := range spanTests {
		sm, _, err := db.EpochSpansMap(ctx, tt.epoch)
		require.NoError(t, err, "Failed to get validator span map")
		require.DeepEqual(t, tt.spanMap, sm, "Get should return validator span map")
	}
}

func TestStore_ReadWriteEpochsSpanByValidatorsIndices(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()
	db.spanCacheEnabled = false

	for _, tt := range spanTests {
		err := db.SaveEpochSpansMap(ctx, tt.epoch, tt.spanMap)
		require.NoError(t, err, "Save validator span map failed")
	}
	res, err := db.EpochsSpanByValidatorsIndices(ctx, []uint64{1, 2, 3}, 3)
	require.NoError(t, err)
	assert.Equal(t, len(spanTests), len(res), "Unexpected number of elements")
	for _, tt := range spanTests {
		assert.DeepEqual(t, tt.spanMap, res[tt.epoch], "Unexpected span map")
	}
	db1 := setupDB(t)
	require.NoError(t, db1.SaveEpochsSpanByValidatorsIndices(ctx, res))
	res, err = db1.EpochsSpanByValidatorsIndices(ctx, []uint64{1, 2, 3}, 3)
	require.NoError(t, err)
	assert.Equal(t, len(spanTests), len(res), "Unexpected number of elements")
	for _, tt := range spanTests {
		assert.DeepEqual(t, tt.spanMap, res[tt.epoch], "Unexpected span map")
	}
}
