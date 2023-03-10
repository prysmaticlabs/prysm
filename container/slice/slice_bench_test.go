package slice

import (
	"math/rand"
	"testing"
)

var largeBenchArrays [][]uint64
var smallBenchArrays [][]uint64

func init() {
	arr0 := []uint64{}
	arr1 := []uint64{}
	for i := 0; i < 10000; i++ {
		arr0 = append(arr0, uint64(i))
		arr1 = append(arr0, rand.Uint64())
	}
	largeBenchArrays = append(largeBenchArrays, arr0)
	largeBenchArrays = append(largeBenchArrays, arr1)

	smallBenchArrays = append(smallBenchArrays, arr0[:100])
	smallBenchArrays = append(smallBenchArrays, arr1[:100])
}

func BenchmarkGenericSliceSubset(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SubsetUint64(largeBenchArrays[0], largeBenchArrays[1])
	}
}

func BenchmarkUint64SliceSubset(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SubsetUint64_old(largeBenchArrays[0], largeBenchArrays[1])
	}
}

func BenchmarkGenericSliceIntersection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IntersectionUint64(largeBenchArrays[0], largeBenchArrays[1])
	}
}
func BenchmarkUint64SliceIntersection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IntersectionUint64_old(largeBenchArrays[0], largeBenchArrays[1])
	}
}
func BenchmarkGenericSliceIntersectionSame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IntersectionUint64(largeBenchArrays[0], largeBenchArrays[0])
	}
}

func BenchmarkUint64SliceIntersectionSame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IntersectionUint64_old(largeBenchArrays[0], largeBenchArrays[0])
	}
}
func BenchmarkGenericSliceUnion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnionUint64(largeBenchArrays[0], largeBenchArrays[1])
	}
}
func BenchmarkUint64SliceUnion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnionUint64_old(largeBenchArrays[0], largeBenchArrays[1])
	}
}
func BenchmarkGenericSliceUnionSame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnionUint64(largeBenchArrays[0], largeBenchArrays[0])
	}
}

func BenchmarkUint64SliceUnionSame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UnionUint64_old(largeBenchArrays[0], largeBenchArrays[0])
	}
}
func BenchmarkGenericSliceSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SetUint64(largeBenchArrays[0])
	}
}
func BenchmarkUint64SliceSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SetUint64_old(largeBenchArrays[0])
	}
}
func BenchmarkGenericSliceNot(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NotUint64(largeBenchArrays[0], largeBenchArrays[1])
	}
}
func BenchmarkUint64SliceNot(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NotUint64_old(largeBenchArrays[0], largeBenchArrays[1])
	}
}
func BenchmarkGenericSliceNotSame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NotUint64(largeBenchArrays[0], largeBenchArrays[0])
	}
}

func BenchmarkUint64SliceNotSame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NotUint64_old(largeBenchArrays[0], largeBenchArrays[0])
	}
}
func BenchmarkGenericSliceIsInLateExit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsInUint64(10000, largeBenchArrays[0])
	}
}
func BenchmarkUint64SliceIsInLateExit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsInUint64_old(10000, largeBenchArrays[1])
	}
}
