package types_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSpan_Marshal_Unmarshal(t *testing.T) {
	span := types.Span{
		MinSpan:     65535,
		MaxSpan:     65535,
		SigBytes:    [2]byte{255, 255},
		HasAttested: true,
	}
	marshaled := span.Marshal()
	unmarshaled, err := types.UnmarshalSpan(marshaled)
	require.NoError(t, err)
	require.DeepEqual(t, span, unmarshaled, "expected unmarshaled to be equal")
}

func BenchmarkSpan_Marshal(b *testing.B) {
	span := types.Span{
		MinSpan:     40,
		MaxSpan:     60,
		SigBytes:    [2]byte{5, 10},
		HasAttested: true,
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = span.Marshal()
	}
}

func BenchmarkSpan_Unmarshal(b *testing.B) {
	span := types.Span{
		MinSpan:     40,
		MaxSpan:     60,
		SigBytes:    [2]byte{5, 10},
		HasAttested: true,
	}
	marshaled := span.Marshal()
	b.ReportAllocs()
	var err error
	for i := 0; i < b.N; i++ {
		span, err = types.UnmarshalSpan(marshaled)
		require.NoError(b, err)
	}
}
