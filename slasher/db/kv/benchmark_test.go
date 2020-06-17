package kv

import (
	"context"
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/urfave/cli/v2"
)

const (
	benchmarkValidator = 300000
)

func BenchmarkStore_SaveEpochSpans(b *testing.B) {
	ctx := context.Background()
	sigBytes := [2]byte{}
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(b, cli.NewContext(&app, set, nil))
	es := &types.EpochStore{}

	es, err := es.SetValidatorSpan(benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.SaveEpochSpans(ctx, uint64(i%54000), es, false)
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
	es := &types.EpochStore{}
	es, err := es.SetValidatorSpan(benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.Log(len(es.Bytes()))
	for i := 0; i < 200; i++ {
		err := db.SaveEpochSpans(ctx, uint64(i), es, false)
		if err != nil {
			b.Fatalf("Save validator span map failed: %v", err)
		}
	}
	b.Log(db.db.Info())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.EpochSpans(ctx, uint64(i%200), false)
		if err != nil {
			b.Fatalf("Read validator span map failed: %v", err)
		}
	}
}

func BenchmarkStore_GetValidatorSpan(b *testing.B) {
	sigBytes := [2]byte{}
	es := &types.EpochStore{}
	es, err := es.SetValidatorSpan(benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), types.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.Log(len(es.Bytes()))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := es.GetValidatorSpan(uint64(i % benchmarkValidator))
		if err != nil {
			b.Fatalf("Read validator span map failed: %v", err)
		}
	}
}

func BenchmarkStore_SetValidatorSpan(b *testing.B) {
	sigBytes := [2]byte{}
	var err error
	es := &types.EpochStore{}
	es, err = es.SetValidatorSpan(benchmarkValidator, types.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), types.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Error(err)
		}
	}
	b.Log(len(es.Bytes()))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es, err = es.SetValidatorSpan(uint64(i%benchmarkValidator), types.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		if err != nil {
			b.Fatalf("Read validator span map failed: %v", err)
		}
	}
}
