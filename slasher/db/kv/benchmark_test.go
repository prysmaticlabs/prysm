package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

const (
	benchmarkValidator = 300000
)

func BenchmarkStore_SaveEpochSpans(b *testing.B) {
	ctx := context.Background()
	sigBytes := [2]byte{}
	db := setupDB(b)
	es := &slashertypes.EpochStore{}

	es, err := es.SetValidatorSpan(benchmarkValidator, slashertypes.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	assert.NoError(b, err)
	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), slashertypes.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
		assert.NoError(b, err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.SaveEpochSpans(ctx, types.Epoch(i%54000), es, false)
		require.NoError(b, err, "Save validator span map failed")
	}
}

func BenchmarkStore_EpochSpans(b *testing.B) {
	db := setupDB(b)
	ctx := context.Background()
	sigBytes := [2]byte{}
	es := &slashertypes.EpochStore{}
	es, err := es.SetValidatorSpan(benchmarkValidator, slashertypes.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	assert.NoError(b, err)
	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), slashertypes.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
		assert.NoError(b, err)
	}
	b.Log(len(es.Bytes()))
	for i := 0; i < 200; i++ {
		err := db.SaveEpochSpans(ctx, types.Epoch(i), es, false)
		require.NoError(b, err, "Save validator span map failed")
	}
	b.Log(db.db.Info())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := db.EpochSpans(ctx, types.Epoch(i%200), false)
		require.NoError(b, err, "Read validator span map failed")
	}
}

func BenchmarkStore_GetValidatorSpan(b *testing.B) {
	sigBytes := [2]byte{}
	es := &slashertypes.EpochStore{}
	es, err := es.SetValidatorSpan(benchmarkValidator, slashertypes.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	assert.NoError(b, err)
	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), slashertypes.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		assert.NoError(b, err)
	}
	b.Log(len(es.Bytes()))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := es.GetValidatorSpan(uint64(i % benchmarkValidator))
		require.NoError(b, err, "Read validator span map failed")
	}
}

func BenchmarkStore_SetValidatorSpan(b *testing.B) {
	sigBytes := [2]byte{}
	var err error
	es := &slashertypes.EpochStore{}
	es, err = es.SetValidatorSpan(benchmarkValidator, slashertypes.Span{MinSpan: 1, MaxSpan: 2, SigBytes: sigBytes, HasAttested: true})
	assert.NoError(b, err)

	for i := 0; i < benchmarkValidator; i++ {
		es, err = es.SetValidatorSpan(uint64(i), slashertypes.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		assert.NoError(b, err)
	}
	b.Log(len(es.Bytes()))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es, err = es.SetValidatorSpan(uint64(i%benchmarkValidator), slashertypes.Span{MinSpan: uint16(i), MaxSpan: uint16(benchmarkValidator - i), SigBytes: sigBytes, HasAttested: true})
		require.NoError(b, err, "Read validator span map failed")
	}
}
