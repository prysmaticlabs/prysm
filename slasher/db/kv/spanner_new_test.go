package kv

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"reflect"
	"testing"

	dbTypes "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/urfave/cli/v2"
)

type spansTestStruct struct {
	name           string
	epoch          uint64
	spansHex       string
	spansResultHex string
	validator1Span types.Span
	err            error
}

var spanNewTests []spansTestStruct

func init() {
	spanNewTests = []spansTestStruct{
		{
			name:           "span too small",
			epoch:          1,
			spansHex:       "00000000",
			spansResultHex: "",
			validator1Span: types.Span{},
			err:            types.ErrWrongSize,
		},
		{
			name:           "no validator 1 in spans",
			epoch:          2,
			spansHex:       "00000000000000",
			spansResultHex: "00000000000000",
			validator1Span: types.Span{},
			err:            nil,
		},
		{
			name:           "validator 1 in spans",
			epoch:          3,
			spansHex:       "0000000000000001000000000000",
			spansResultHex: "0000000000000001000000000000",
			validator1Span: types.Span{MinSpan: 1},
			err:            nil,
		},
	}

}

func TestValidatorSpans_NilDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	validatorIdx := uint64(1)
	es, err := db.EpochSpans(ctx, validatorIdx, false)
	if err != nil {
		t.Fatalf("Nil EpochSpansMap should not return error: %v", err)
	}
	cleanStore, err := types.NewEpochStore([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(es, cleanStore) {
		t.Fatal("EpochSpans should return empty byte array if no record exists in the db")
	}
}

func TestStore_SaveReadEpochSpans(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	for _, tt := range spanNewTests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := hex.DecodeString(tt.spansHex)
			if err != nil {
				t.Fatal(err)
			}
			es, err := types.NewEpochStore(spans)
			if err != tt.err {
				t.Fatalf("Failed to get the right error expected: %v got: %v", tt.err, err)
			}
			if err = db.SaveEpochSpans(ctx, tt.epoch, es, false); err != nil {
				t.Fatal(err)
			}
			sm, err := db.EpochSpans(ctx, tt.epoch, false)
			if err != nil {
				t.Fatalf("Failed to get validator spans: %v", err)
			}
			spansResult, err := hex.DecodeString(tt.spansResultHex)
			if err != nil {
				t.Fatal(err)
			}
			esr, err := types.NewEpochStore(spansResult)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(sm, esr) {
				t.Fatalf("Get should return validator spans: %v got: %v", spansResult, sm)
			}

			s, err := es.GetValidatorSpan(1)
			if err != nil {
				t.Fatalf("Failed to get validator 1 span: %v", err)
			}
			if !reflect.DeepEqual(s, tt.validator1Span) {
				t.Fatalf("Get should return validator span for validator 2: %v got: %v", tt.validator1Span, s)
			}
		})
	}
}

func TestStore_SaveEpochSpans_ToCache(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	spansToSave := map[uint64]types.Span{
		0:     {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
		10:    {MinSpan: 43, MaxSpan: 32, SigBytes: [2]byte{10, 13}, HasAttested: true},
		1000:  {MinSpan: 40, MaxSpan: 36, SigBytes: [2]byte{61, 151}, HasAttested: false},
		10000: {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
		50000: {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
		100:   {MinSpan: 49, MaxSpan: 96, SigBytes: [2]byte{11, 98}, HasAttested: true},
	}
	epochStore, err := types.EpochStoreFromMap(spansToSave)
	if err != nil {
		t.Fatal(err)
	}

	epoch := uint64(9)
	if err := db.SaveEpochSpans(ctx, epoch, epochStore, dbTypes.UseCache); err != nil {
		t.Fatal(err)
	}

	esFromCache, err := db.EpochSpans(ctx, epoch, dbTypes.UseCache)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(epochStore.Bytes(), esFromCache.Bytes()) {
		t.Fatalf("Expected store from DB to be %#x, received %#x", epochStore.Bytes(), esFromCache.Bytes())
	}

	esFromDB, err := db.EpochSpans(ctx, epoch, dbTypes.UseDB)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(esFromDB.Bytes(), esFromCache.Bytes()) {
		t.Fatalf("Expected store asked from DB to use cache, \nreceived %#x, \nexpected %#x", esFromDB.Bytes(), esFromCache.Bytes())
	}
}

func TestStore_SaveEpochSpans_ToDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	spansToSave := map[uint64]types.Span{
		0:      {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
		10:     {MinSpan: 43, MaxSpan: 32, SigBytes: [2]byte{10, 13}, HasAttested: true},
		1000:   {MinSpan: 40, MaxSpan: 36, SigBytes: [2]byte{61, 151}, HasAttested: false},
		10000:  {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
		100000: {MinSpan: 20, MaxSpan: 64, SigBytes: [2]byte{170, 215}, HasAttested: false},
		100:    {MinSpan: 49, MaxSpan: 96, SigBytes: [2]byte{11, 98}, HasAttested: true},
	}
	epochStore, err := types.EpochStoreFromMap(spansToSave)
	if err != nil {
		t.Fatal(err)
	}

	epoch := uint64(9)
	if err := db.SaveEpochSpans(ctx, epoch, epochStore, dbTypes.UseDB); err != nil {
		t.Fatal(err)
	}

	// Expect cache to retrieve from DB if its not in cache.
	esFromCache, err := db.EpochSpans(ctx, epoch, dbTypes.UseCache)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(esFromCache.Bytes(), epochStore.Bytes()) {
		t.Fatalf("Expected cache request to be %#x, expected %#x", epochStore.Bytes(), esFromCache.Bytes())
	}

	esFromDB, err := db.EpochSpans(ctx, epoch, dbTypes.UseDB)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(epochStore.Bytes(), esFromDB.Bytes()) {
		t.Fatalf("Expected store from DB to be %#x, received %#x", epochStore.Bytes(), esFromDB.Bytes())
	}
}
