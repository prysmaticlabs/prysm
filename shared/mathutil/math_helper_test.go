package mathutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestIntegerSquareRoot(t *testing.T) {
	tt := []struct {
		number uint64
		root   uint64
	}{
		{
			number: 20,
			root:   4,
		},
		{
			number: 200,
			root:   14,
		},
		{
			number: 1987,
			root:   44,
		},
		{
			number: 34989843,
			root:   5915,
		},
		{
			number: 97282,
			root:   311,
		},
		{
			number: 1 << 32,
			root:   1 << 16,
		},
		{
			number: (1 << 32) + 1,
			root:   1 << 16,
		},
		{
			number: 1 << 33,
			root:   92681,
		},
		{
			number: 1 << 60,
			root:   1 << 30,
		},
		{
			number: 1 << 53,
			root:   94906265,
		},
		{
			number: 1 << 62,
			root:   1 << 31,
		},
		{
			number: 1024,
			root:   32,
		},
		{
			number: 4,
			root:   2,
		},
		{
			number: 16,
			root:   4,
		},
	}

	for _, testVals := range tt {
		require.Equal(t, testVals.root, mathutil.IntegerSquareRoot(testVals.number))
	}
}

func BenchmarkIntegerSquareRoot(b *testing.B) {
	val := uint64(1 << 62)
	for i := 0; i < b.N; i++ {
		require.Equal(b, 1<<31, mathutil.IntegerSquareRoot(val))
	}
}

func BenchmarkIntegerSquareRoot_WithDatatable(b *testing.B) {
	val := uint64(1024)
	for i := 0; i < b.N; i++ {
		require.Equal(b, 32, mathutil.IntegerSquareRoot(val))
	}
}

func TestCeilDiv8(t *testing.T) {
	tests := []struct {
		number int
		div8   int
	}{
		{
			number: 20,
			div8:   3,
		},
		{
			number: 200,
			div8:   25,
		},
		{
			number: 1987,
			div8:   249,
		},
		{
			number: 1,
			div8:   1,
		},
		{
			number: 97282,
			div8:   12161,
		},
	}

	for _, tt := range tests {
		require.Equal(t, tt.div8, mathutil.CeilDiv8(tt.number))
	}
}

func TestIsPowerOf2(t *testing.T) {
	tests := []struct {
		a uint64
		b bool
	}{
		{
			a: 2,
			b: true,
		},
		{
			a: 64,
			b: true,
		},
		{
			a: 100,
			b: false,
		},
		{
			a: 1024,
			b: true,
		},
		{
			a: 0,
			b: false,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.b, mathutil.IsPowerOf2(tt.a))
	}
}

func TestPowerOf2(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
	}{
		{
			a: 3,
			b: 8,
		},
		{
			a: 20,
			b: 1048576,
		},
		{
			a: 11,
			b: 2048,
		},
		{
			a: 8,
			b: 256,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.b, mathutil.PowerOf2(tt.a))
	}
}

func TestClosestPowerOf2(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
	}{
		{
			a: 10,
			b: 8,
		},
		{
			a: 300,
			b: 256,
		},
		{
			a: 1200,
			b: 1024,
		},
		{
			a: 4500,
			b: 4096,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.b, mathutil.ClosestPowerOf2(tt.a))
	}
}

func TestMaxValue(t *testing.T) {
	tests := []struct {
		a      uint64
		b      uint64
		result uint64
	}{
		{
			a:      10,
			b:      8,
			result: 10,
		},
		{
			a:      300,
			b:      256,
			result: 300,
		},
		{
			a:      1200,
			b:      1024,
			result: 1200,
		},
		{
			a:      4500,
			b:      4096,
			result: 4500,
		},
		{
			a:      9999,
			b:      9999,
			result: 9999,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.result, mathutil.Max(tt.a, tt.b))
	}
}

func TestMinValue(t *testing.T) {
	tests := []struct {
		a      uint64
		b      uint64
		result uint64
	}{
		{
			a:      10,
			b:      8,
			result: 8,
		},
		{
			a:      300,
			b:      256,
			result: 256,
		},
		{
			a:      1200,
			b:      1024,
			result: 1024,
		},
		{
			a:      4500,
			b:      4096,
			result: 4096,
		},
		{
			a:      9999,
			b:      9999,
			result: 9999,
		},
	}
	for _, tt := range tests {
		require.Equal(t, tt.result, mathutil.Min(tt.a, tt.b))
	}
}
