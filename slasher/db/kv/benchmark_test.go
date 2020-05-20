package kv

import (
	"context"
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"gopkg.in/urfave/cli.v2"
)

var input [][]byte

const (
	benchmarkValidator = 300000
)

func init() {

}

func BenchmarkStore_SaveEpochSpans(b *testing.B) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(b, cli.NewContext(&app, set, nil))
	ctx := context.Background()
	sigBytes := [2]byte{}
	spans, err := db.SetValidatorSpan(ctx, []byte{}, benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		spans, err = db.SetValidatorSpan(ctx, spans, uint64(i), types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.SaveEpochSpans(ctx, uint64(i%54000), spans)
		if err != nil {
			b.Fatalf("Save validator span map failed: %v", err)
		}
	}

}

func BenchmarkStore_EpochSpans(b *testing.B) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(b, cli.NewContext(&app, set, nil))
	ctx := context.Background()
	sigBytes := [2]byte{}
	spans, err := db.SetValidatorSpan(ctx, []byte{}, benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		spans, err = db.SetValidatorSpan(ctx, spans, uint64(i), types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.Log(len(spans))
	for i := 0; i < 200; i++ {
		err := db.SaveEpochSpans(ctx, uint64(i), spans)
		if err != nil {
			b.Fatalf("Save validator span map failed: %v", err)
		}
	}
	b.Log(db.db.Info())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := db.EpochSpans(ctx, uint64(i%200))
		if err != nil {
			b.Fatalf("Read validator span map failed: %v", err)
		}

	}

}

func BenchmarkStore_GetValidatorSpan(b *testing.B) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(b, cli.NewContext(&app, set, nil))
	ctx := context.Background()
	sigBytes := [2]byte{}
	spans, err := db.SetValidatorSpan(ctx, []byte{}, benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		spans, err = db.SetValidatorSpan(ctx, spans, uint64(i), types.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.Log(len(spans))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.GetValidatorSpan(ctx, spans, uint64(i%benchmarkValidator))
		if err != nil {
			b.Fatalf("Read validator span map failed: %v", err)
		}

	}

}

func BenchmarkStore_SetValidatorSpan(b *testing.B) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(b, cli.NewContext(&app, set, nil))
	ctx := context.Background()
	sigBytes := [2]byte{}
	spans, err := db.SetValidatorSpan(ctx, []byte{}, benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		spans, err = db.SetValidatorSpan(ctx, spans, uint64(i), types.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.Log(len(spans))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.SetValidatorSpan(ctx, spans, uint64(i%benchmarkValidator), types.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Fatalf("Read validator span map failed: %v", err)
		}

	}

}
