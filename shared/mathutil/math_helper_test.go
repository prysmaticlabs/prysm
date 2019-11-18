package mathutil_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/mathutil"
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
		root := mathutil.IntegerSquareRoot(testVals.number)
		if testVals.root != root {
			t.Errorf("For %d, expected root and computed root are not equal want %d, got %d", testVals.number, testVals.root, root)
		}
	}
}

func BenchmarkIntegerSquareRoot(b *testing.B) {
	val := uint64(1 << 62)
	for i := 0; i < b.N; i++ {
		root := mathutil.IntegerSquareRoot(val)
		if root != 1<<31 {
			b.Fatalf("Expected root and computed root are not equal 1<<31, %d", root)
		}
	}
}

func BenchmarkIntegerSquareRoot_WithDatatable(b *testing.B) {
	val := uint64(1024)
	for i := 0; i < b.N; i++ {
		root := mathutil.IntegerSquareRoot(val)
		if root != 32 {
			b.Fatalf("Expected root and computed root are not equal 32, %d", root)
		}
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
		div8 := mathutil.CeilDiv8(tt.number)
		if tt.div8 != div8 {
			t.Fatalf("Div8 was not an expected value. Wanted: %d, got: %d", tt.div8, div8)
		}
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
		if tt.b != mathutil.IsPowerOf2(tt.a) {
			t.Fatalf("IsPowerOf2(%d) = %v, wanted: %v", tt.a, mathutil.IsPowerOf2(tt.a), tt.b)
		}
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
		if tt.b != mathutil.PowerOf2(tt.a) {
			t.Fatalf("PowerOf2(%d) = %d, wanted: %d", tt.a, mathutil.PowerOf2(tt.a), tt.b)
		}
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
		if tt.b != mathutil.ClosestPowerOf2(tt.a) {
			t.Fatalf("ClosestPowerOf2(%d) = %d, wanted: %d", tt.a, mathutil.ClosestPowerOf2(tt.a), tt.b)
		}
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
		if tt.result != mathutil.Max(tt.a, tt.b) {
			t.Fatalf("Max(%d) = %d, wanted: %d", tt.a, mathutil.Max(tt.a, tt.b), tt.result)
		}
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
		if tt.result != mathutil.Min(tt.a, tt.b) {
			t.Fatalf("Min(%d) = %d, wanted: %d", tt.a, mathutil.Min(tt.a, tt.b), tt.result)
		}
	}
}
