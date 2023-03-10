package slice

import (
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// SubsetUint64 returns true if the first array is
// completely contained in the second array with time
// complexity of approximately o(n).
func SubsetUint64(a, b []uint64) bool {
	return Subset(a, b)
}

// IntersectionUint64 of any number of uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionUint64(s ...[]uint64) []uint64 {
	return Intersection(s...)
}

// UnionUint64 of any number of uint64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionUint64(s ...[]uint64) []uint64 {
	return Union(s...)
}

// SetUint64 returns a slice with only unique
// values from the provided list of indices.
func SetUint64(a []uint64) []uint64 {
	return Set(a)
}

// IsUint64Sorted verifies if a uint64 slice is sorted in ascending order.
func IsUint64Sorted(a []uint64) bool {
	return IsSorted(a)
}

// NotUint64 returns the uint64 in slice b that are
// not in slice a with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func NotUint64(a, b []uint64) []uint64 {
	return Not(a, b)
}

// IsInUint64 returns true if a is in b and False otherwise.
func IsInUint64(a uint64, b []uint64) bool {
	return IsIn(a, b)
}

// IntersectionInt64 of any number of int64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionInt64(s ...[]int64) []int64 {
	return Intersection(s...)
}

// UnionInt64 of any number of int64 slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func UnionInt64(s ...[]int64) []int64 {
	return Union(s...)
}

// NotInt64 returns the int64 in slice a that are
// not in slice b with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func NotInt64(a, b []int64) []int64 {
	return Not(a, b)
}

// IsInInt64 returns true if a is in b and False otherwise.
func IsInInt64(a int64, b []int64) bool {
	return IsIn(a, b)
}

// SplitOffset returns the start index of a given list splits into chunks,
// it computes (listsize * index) / chunks.
//
// Spec pseudocode definition:
// def get_split_offset(list_size: int, chunks: int, index: int) -> int:
//
//	"""
//	Returns a value such that for a list L, chunk count k and index i,
//	split(L, k)[i] == L[get_split_offset(len(L), k, i): get_split_offset(len(L), k, i+1)]
//	"""
//	return (list_size * index) // chunks
func SplitOffset(listSize, chunks, index uint64) uint64 {
	return (listSize * index) / chunks
}

// IntersectionSlot of any number of types.Slot slices with time
// complexity of approximately O(n) leveraging a map to
// check for element existence off by a constant factor
// of underlying map efficiency.
func IntersectionSlot(s ...[]primitives.Slot) []primitives.Slot {
	return Intersection(s...)
}

// NotSlot returns the types.Slot in slice b that are
// not in slice a with time complexity of approximately
// O(n) leveraging a map to check for element existence
// off by a constant factor of underlying map efficiency.
func NotSlot(a, b []primitives.Slot) []primitives.Slot {
	return Not(a, b)
}

// IsInSlots returns true if a is in b and False otherwise.
func IsInSlots(a primitives.Slot, b []primitives.Slot) bool {
	return IsIn(a, b)
}
