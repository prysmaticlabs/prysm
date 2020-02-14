package kv

import (
	"context"
	"flag"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/urfave/cli"
)

type spanMapTestStruct struct {
	validatorIdx uint64
	spanMap      *slashpb.EpochSpanMap
}

var spanTests []spanMapTestStruct

func init() {
	spanTests = []spanMapTestStruct{
		{
			validatorIdx: 1,
			spanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 10, MaxEpochSpan: 20},
					2: {MinEpochSpan: 11, MaxEpochSpan: 21},
					3: {MinEpochSpan: 12, MaxEpochSpan: 22},
				},
			},
		},
		{
			validatorIdx: 2,
			spanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 10, MaxEpochSpan: 20},
					2: {MinEpochSpan: 11, MaxEpochSpan: 21},
					3: {MinEpochSpan: 12, MaxEpochSpan: 22},
				},
			},
		},
		{
			validatorIdx: 3,
			spanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 10, MaxEpochSpan: 20},
					2: {MinEpochSpan: 11, MaxEpochSpan: 21},
					3: {MinEpochSpan: 12, MaxEpochSpan: 22},
				},
			},
		},
	}
}

func TestValidatorSpanMap_NilDB(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	validatorIdx := uint64(1)
	vsm, err := db.ValidatorSpansMap(ctx, validatorIdx)
	if err != nil {
		t.Fatalf("Nil ValidatorSpansMap should not return error: %v", err)
	}
	if !reflect.DeepEqual(vsm.EpochSpanMap, map[uint64]*slashpb.MinMaxEpochSpan{}) {
		t.Fatal("ValidatorSpansMap should return nil")
	}
}

func TestValidatorSpanMap_Save(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(ctx, tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
		sm, err := db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}

		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
	}
}

func TestValidatorSpanMap_SaveWithCache(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(ctx, tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
		// wait for value to pass through cache buffers
		time.Sleep(time.Millisecond * 10)
		sm, err := db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}

		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
	}
}

func TestValidatorSpanMap_Delete(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(ctx, tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
	}

	for _, tt := range spanTests {
		sm, err := db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
		err = db.DeleteValidatorSpanMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Delete validator span map error: %v", err)
		}
		sm, err = db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(sm.EpochSpanMap, map[uint64]*slashpb.MinMaxEpochSpan{}) {
			t.Errorf("Expected validator span map to be deleted, received: %v", sm)
		}
	}
}

func TestValidatorSpanMap_DeleteWithCache(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(ctx, tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
	}
	// wait for value to pass through cache buffers
	time.Sleep(time.Millisecond * 10)
	for _, tt := range spanTests {
		sm, err := db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
		err = db.DeleteValidatorSpanMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Delete validator span map error: %v", err)
		}
		// wait for value to pass through cache buffers
		time.Sleep(time.Millisecond * 10)
		sm, err = db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(sm.EpochSpanMap, map[uint64]*slashpb.MinMaxEpochSpan{}) {
			t.Errorf("Expected validator span map to be deleted, received: %v", sm)
		}
	}
}

func TestValidatorSpanMap_SaveOnEvict(t *testing.T) {
	db := setupDBDiffCacheSize(t, 5, 5)
	defer teardownDB(t, db)
	ctx := context.Background()

	tsm := &spanMapTestStruct{
		validatorIdx: 1,
		spanMap: &slashpb.EpochSpanMap{
			EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
				1: {MinEpochSpan: 10, MaxEpochSpan: 20},
				2: {MinEpochSpan: 11, MaxEpochSpan: 21},
				3: {MinEpochSpan: 12, MaxEpochSpan: 22},
			},
		},
	}
	for i := uint64(0); i < 6; i++ {
		err := db.SaveValidatorSpansMap(ctx, i, tsm.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
	}

	// Wait for value to pass through cache buffers.
	time.Sleep(time.Millisecond * 1000)
	for i := uint64(0); i < 6; i++ {
		sm, err := db.ValidatorSpansMap(ctx, i)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tsm.spanMap) {
			t.Fatalf("Get should return validator: %d span map: %v got: %v", i, tsm.spanMap, sm)
		}
	}
}

func TestValidatorSpanMap_SaveCachedSpansMaps(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.Bool(flags.UseSpanCacheFlag.Name, true, "enable span map cache")
	db := setupDB(t, cli.NewContext(app, set, nil))
	defer teardownDB(t, db)
	ctx := context.Background()

	for _, tt := range spanTests {
		err := db.SaveValidatorSpansMap(ctx, tt.validatorIdx, tt.spanMap)
		if err != nil {
			t.Fatalf("Save validator span map failed: %v", err)
		}
	}
	// wait for value to pass through cache buffers
	time.Sleep(time.Millisecond * 10)
	err := db.SaveCachedSpansMaps(ctx)
	if err != nil {
		t.Errorf("Failed to save cached span maps to db: %v", err)
	}
	db.spanCache.Clear()
	for _, tt := range spanTests {
		sm, err := db.ValidatorSpansMap(ctx, tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to get validator span map: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.spanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.spanMap, sm)
		}
	}
}
