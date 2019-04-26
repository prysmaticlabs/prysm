package sliceutil

import (
	"reflect"
	"testing"
)

func TestIntersectionUint64(t *testing.T) {
	testCases := []struct {
		setA []uint64
		setB []uint64
		out  []uint64
	}{
		{[]uint64{2, 3, 5}, []uint64{3}, []uint64{3}},
		{[]uint64{2, 3, 5}, []uint64{3, 5}, []uint64{3, 5}},
		{[]uint64{2, 3, 5}, []uint64{5, 3, 2}, []uint64{5, 3, 2}},
		{[]uint64{2, 3, 5}, []uint64{2, 3, 5}, []uint64{2, 3, 5}},
		{[]uint64{2, 3, 5}, []uint64{}, []uint64{}},
		{[]uint64{}, []uint64{2, 3, 5}, []uint64{}},
		{[]uint64{}, []uint64{}, []uint64{}},
		{[]uint64{1}, []uint64{1}, []uint64{1}},
	}
	for _, tt := range testCases {
		result := IntersectionUint64(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}

	}
}

func TestIntersectionInt64(t *testing.T) {
	testCases := []struct {
		setA []int64
		setB []int64
		out  []int64
	}{
		{[]int64{2, 3, 5}, []int64{3}, []int64{3}},
		{[]int64{2, 3, 5}, []int64{3, 5}, []int64{3, 5}},
		{[]int64{2, 3, 5}, []int64{5, 3, 2}, []int64{5, 3, 2}},
		{[]int64{2, 3, 5}, []int64{2, 3, 5}, []int64{2, 3, 5}},
		{[]int64{2, 3, 5}, []int64{}, []int64{}},
		{[]int64{}, []int64{2, 3, 5}, []int64{}},
		{[]int64{}, []int64{}, []int64{}},
		{[]int64{1}, []int64{1}, []int64{1}},
	}
	for _, tt := range testCases {
		result := IntersectionInt64(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}

	}

}

func TestUnionUint64(t *testing.T) {
	testCases := []struct {
		setA []uint64
		setB []uint64
		out  []uint64
	}{
		{[]uint64{2, 3, 5}, []uint64{4, 6}, []uint64{2, 3, 5, 4, 6}},
		{[]uint64{2, 3, 5}, []uint64{3, 5}, []uint64{2, 3, 5}},
		{[]uint64{2, 3, 5}, []uint64{2, 3, 5}, []uint64{2, 3, 5}},
		{[]uint64{2, 3, 5}, []uint64{}, []uint64{2, 3, 5}},
		{[]uint64{}, []uint64{2, 3, 5}, []uint64{2, 3, 5}},
		{[]uint64{}, []uint64{}, []uint64{}},
		{[]uint64{1}, []uint64{1}, []uint64{1}},
	}
	for _, tt := range testCases {
		result := UnionUint64(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}

	}
}

func TestUnionInt64(t *testing.T) {
	testCases := []struct {
		setA []int64
		setB []int64
		out  []int64
	}{
		{[]int64{2, 3, 5}, []int64{4, 6}, []int64{2, 3, 5, 4, 6}},
		{[]int64{2, 3, 5}, []int64{3, 5}, []int64{2, 3, 5}},
		{[]int64{2, 3, 5}, []int64{2, 3, 5}, []int64{2, 3, 5}},
		{[]int64{2, 3, 5}, []int64{}, []int64{2, 3, 5}},
		{[]int64{}, []int64{2, 3, 5}, []int64{2, 3, 5}},
		{[]int64{}, []int64{}, []int64{}},
		{[]int64{1}, []int64{1}, []int64{1}},
	}
	for _, tt := range testCases {
		result := UnionInt64(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}

	}

}

func TestNotUint64(t *testing.T) {
	testCases := []struct {
		setA []uint64
		setB []uint64
		out  []uint64
	}{
		{[]uint64{4, 6}, []uint64{2, 3, 5, 4, 6}, []uint64{2, 3, 5}},
		{[]uint64{3, 5}, []uint64{2, 3, 5}, []uint64{2}},
		{[]uint64{2, 3, 5}, []uint64{2, 3, 5}, []uint64{}},
		{[]uint64{2}, []uint64{2, 3, 5}, []uint64{3, 5}},
		{[]uint64{}, []uint64{2, 3, 5}, []uint64{2, 3, 5}},
		{[]uint64{}, []uint64{}, []uint64{}},
		{[]uint64{1}, []uint64{1}, []uint64{}},
	}
	for _, tt := range testCases {
		result := NotUint64(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func TestNotInt64(t *testing.T) {
	testCases := []struct {
		setA []int64
		setB []int64
		out  []int64
	}{
		{[]int64{4, 6}, []int64{2, 3, 5, 4, 6}, []int64{2, 3, 5}},
		{[]int64{3, 5}, []int64{2, 3, 5}, []int64{2}},
		{[]int64{2, 3, 5}, []int64{2, 3, 5}, []int64{}},
		{[]int64{2}, []int64{2, 3, 5}, []int64{3, 5}},
		{[]int64{}, []int64{2, 3, 5}, []int64{2, 3, 5}},
		{[]int64{}, []int64{}, []int64{}},
		{[]int64{1}, []int64{1}, []int64{}},
	}
	for _, tt := range testCases {
		result := NotInt64(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func TestIsInUint64(t *testing.T) {
	testCases := []struct {
		a      uint64
		b      []uint64
		result bool
	}{
		{0, []uint64{}, false},
		{0, []uint64{0}, true},
		{4, []uint64{2, 3, 5, 4, 6}, true},
		{100, []uint64{2, 3, 5, 4, 6}, false},
	}
	for _, tt := range testCases {
		result := IsInUint64(tt.a, tt.b)
		if result != tt.result {
			t.Errorf("IsIn(%d, %v)=%v, wanted: %v",
				tt.a, tt.b, result, tt.result)
		}
	}
}

func TestIsInInt64(t *testing.T) {
	testCases := []struct {
		a      int64
		b      []int64
		result bool
	}{
		{0, []int64{}, false},
		{0, []int64{0}, true},
		{4, []int64{2, 3, 5, 4, 6}, true},
		{100, []int64{2, 3, 5, 4, 6}, false},
	}
	for _, tt := range testCases {
		result := IsInInt64(tt.a, tt.b)
		if result != tt.result {
			t.Errorf("IsIn(%d, %v)=%v, wanted: %v",
				tt.a, tt.b, result, tt.result)
		}
	}
}
