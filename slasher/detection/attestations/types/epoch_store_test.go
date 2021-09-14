package types_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	dbTypes "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestEpochStore_GetValidatorSpan_Format(t *testing.T) {
	type formatTest struct {
		name         string
		hexToDecode  string
		expectedErr  error
		expectedSpan map[uint64]types.Span
	}
	tests := []formatTest{
		{
			name:         "too small",
			hexToDecode:  "000000",
			expectedErr:  types.ErrWrongSize,
			expectedSpan: nil,
		},
		{
			name:         "too big",
			hexToDecode:  "0000000000000000",
			expectedErr:  types.ErrWrongSize,
			expectedSpan: nil,
		},
		{
			name:        "one validator",
			hexToDecode: "01010101010101",
			expectedErr: nil,
			expectedSpan: map[uint64]types.Span{
				0: {MinSpan: 257, MaxSpan: 257, SigBytes: [2]byte{1, 1}, HasAttested: true},
				1: {},
			},
		},
		{
			name:        "two validators",
			hexToDecode: "1181019551010001010114770101",
			expectedErr: nil,
			expectedSpan: map[uint64]types.Span{
				0: {MinSpan: 33041, MaxSpan: 38145, SigBytes: [2]byte{81, 1}, HasAttested: false},
				1: {MinSpan: 257, MaxSpan: 5121, SigBytes: [2]byte{119, 1}, HasAttested: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decodedHex, err := hex.DecodeString(tt.hexToDecode)
			require.NoError(t, err)
			es, err := types.NewEpochStore(decodedHex)
			if tt.expectedErr != nil {
				require.ErrorContains(t, tt.expectedErr.Error(), err)
				return
			} else {
				require.NoError(t, err)
			}
			span0, err := es.GetValidatorSpan(0)
			assert.NoError(t, err)
			assert.DeepEqual(t, tt.expectedSpan[0], span0, "Unexpected span")
			span1, err := es.GetValidatorSpan(1)
			assert.NoError(t, err)
			assert.DeepEqual(t, tt.expectedSpan[1], span1, "Unexpected span")
		})
	}
}

func TestEpochStore_GetValidatorSpan_Matches(t *testing.T) {
	type matchTest struct {
		name       string
		spanMap    map[uint64]types.Span
		highestIdx uint64
	}
	tests := []matchTest{
		{
			name: "5 vals",
			spanMap: map[uint64]types.Span{
				0: {MinSpan: 5, MaxSpan: 66, SigBytes: [2]byte{43, 29}, HasAttested: true},
				1: {MinSpan: 53, MaxSpan: 31, SigBytes: [2]byte{12, 93}, HasAttested: false},
				3: {MinSpan: 40, MaxSpan: 34, SigBytes: [2]byte{66, 255}, HasAttested: false},
				4: {MinSpan: 20, MaxSpan: 64, SigBytes: [2]byte{199, 255}, HasAttested: true},
				2: {MinSpan: 59, MaxSpan: 99, SigBytes: [2]byte{18, 98}, HasAttested: true},
			},
			highestIdx: 4,
		},
		{
			name: "5 vals, 5 apart",
			spanMap: map[uint64]types.Span{
				0:  {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 29}, HasAttested: false},
				5:  {MinSpan: 13, MaxSpan: 32, SigBytes: [2]byte{10, 93}, HasAttested: true},
				20: {MinSpan: 90, MaxSpan: 64, SigBytes: [2]byte{190, 225}, HasAttested: true},
				15: {MinSpan: 70, MaxSpan: 36, SigBytes: [2]byte{60, 252}, HasAttested: false},
				10: {MinSpan: 39, MaxSpan: 96, SigBytes: [2]byte{10, 98}, HasAttested: true},
			},
			highestIdx: 20,
		},
		{
			name: "random vals",
			spanMap: map[uint64]types.Span{
				0:      {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
				10:     {MinSpan: 43, MaxSpan: 32, SigBytes: [2]byte{10, 13}, HasAttested: true},
				1000:   {MinSpan: 40, MaxSpan: 36, SigBytes: [2]byte{61, 151}, HasAttested: false},
				100000: {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{110, 225}, HasAttested: true},
				10000:  {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
				100:    {MinSpan: 49, MaxSpan: 96, SigBytes: [2]byte{11, 98}, HasAttested: true},
			},
			highestIdx: 100000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es, err := types.EpochStoreFromMap(tt.spanMap)
			require.NoError(t, err)
			require.Equal(t, tt.highestIdx, es.HighestObservedIdx(), "Unexpected highest index")
			for k, v := range tt.spanMap {
				span, err := es.GetValidatorSpan(k)
				require.NoError(t, err)
				require.DeepEqual(t, v, span, "Unexpected span")
			}
		})
	}
}

func TestEpochStore_SetValidatorSpan(t *testing.T) {
	type matchTest struct {
		name         string
		spanMapToAdd map[uint64]types.Span
		resultMap    map[uint64]types.Span
	}
	tests := []matchTest{
		{
			name:         "create",
			spanMapToAdd: map[uint64]types.Span{},
			resultMap: map[uint64]types.Span{
				0:    {},
				16:   {},
				200:  {},
				1000: {},
			},
		},
		{
			name: "add val idx 100 ",
			spanMapToAdd: map[uint64]types.Span{
				100: {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
			},
			resultMap: map[uint64]types.Span{
				0:    {},
				16:   {},
				100:  {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
				200:  {},
				1000: {},
			},
		},
		{
			name: "add val idx 1000",
			spanMapToAdd: map[uint64]types.Span{
				1000: {MinSpan: 53, MaxSpan: 122, SigBytes: [2]byte{200, 119}, HasAttested: true},
			},
			resultMap: map[uint64]types.Span{
				0:    {},
				16:   {},
				100:  {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
				200:  {},
				1000: {MinSpan: 53, MaxSpan: 122, SigBytes: [2]byte{200, 119}, HasAttested: true},
			},
		},
		{
			name: "add val idx 1000",
			spanMapToAdd: map[uint64]types.Span{
				1500: {MinSpan: 3, MaxSpan: 12, SigBytes: [2]byte{0, 1}, HasAttested: true},
				50:   {MinSpan: 50, MaxSpan: 102, SigBytes: [2]byte{200, 19}, HasAttested: false},
			},
			resultMap: map[uint64]types.Span{
				0:    {},
				16:   {},
				50:   {MinSpan: 50, MaxSpan: 102, SigBytes: [2]byte{200, 19}, HasAttested: false},
				100:  {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
				200:  {},
				1000: {MinSpan: 53, MaxSpan: 122, SigBytes: [2]byte{200, 119}, HasAttested: true},
				1500: {MinSpan: 3, MaxSpan: 12, SigBytes: [2]byte{0, 1}, HasAttested: true},
			},
		},
	}
	es, err := types.NewEpochStore([]byte{})
	require.NoError(t, err)
	require.Equal(t, uint64(0), es.HighestObservedIdx(), "Expected highest index to be 0")
	lastIdx := uint64(0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.spanMapToAdd {
				es, err = es.SetValidatorSpan(k, v)
				require.NoError(t, err)
				if k > lastIdx {
					lastIdx = k
				}
			}
			for k, v := range tt.resultMap {
				span, err := es.GetValidatorSpan(k)
				require.NoError(t, err)
				assert.DeepEqual(t, v, span, "Unexpected span")
			}
			require.Equal(t, lastIdx, es.HighestObservedIdx(), "Unexpected highest index")
		})
	}
}

func BenchmarkEpochStore_Save(b *testing.B) {
	amount := uint64(100000)
	store, _ := generateEpochStore(b, amount)

	b.Run(fmt.Sprintf("%d new", amount), func(b *testing.B) {
		db := testDB.SetupSlasherDB(b, false)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			require.NoError(b, db.SaveEpochSpans(context.Background(), 1, store, dbTypes.UseDB))
		}
	})
}

func generateEpochStore(t testing.TB, n uint64) (*types.EpochStore, map[uint64]types.Span) {
	epochStore, err := types.NewEpochStore([]byte{})
	require.NoError(t, err)
	spanMap := make(map[uint64]types.Span)
	for i := uint64(0); i < n; i++ {
		span := types.Span{
			MinSpan:     14,
			MaxSpan:     8,
			SigBytes:    [2]byte{5, 13},
			HasAttested: true,
		}
		spanMap[i] = span
		epochStore, err = epochStore.SetValidatorSpan(i, span)
		require.NoError(t, err)
	}
	return epochStore, spanMap
}
