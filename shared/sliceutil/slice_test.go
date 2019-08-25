package sliceutil

import (
	"reflect"
	"testing"
)

func TestSubsetUint64(t *testing.T) {
	testCases := []struct {
		setA []uint64
		setB []uint64
		out  bool
	}{
		{[]uint64{1}, []uint64{1, 2, 3, 4}, true},
		{[]uint64{1, 2, 3, 4}, []uint64{1, 2, 3, 4}, true},
		{[]uint64{1, 1}, []uint64{1, 2, 3, 4}, false},
		{[]uint64{}, []uint64{1}, true},
		{[]uint64{1}, []uint64{}, false},
		{[]uint64{1, 2, 3, 4, 5}, []uint64{1, 2, 3, 4}, false},
	}
	for _, tt := range testCases {
		result := SubsetUint64(tt.setA, tt.setB)
		if result != tt.out {
			t.Errorf("%v, got %v, want %v", tt.setA, result, tt.out)
		}
	}
}

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

func TestIsSortedUint64(t *testing.T) {
	testCases := []struct {
		setA []uint64
		out  bool
	}{
		{[]uint64{1, 2, 3}, true},
		{[]uint64{3, 1, 3}, false},
		{[]uint64{1}, true},
		{[]uint64{}, true},
	}
	for _, tt := range testCases {
		result := IsUint64Sorted(tt.setA)
		if result != tt.out {
			t.Errorf("got %v, want %v", result, tt.out)
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
	items := [][]uint64{
		{3, 4, 5},
		{6, 7, 8},
		{9, 10, 11},
	}
	variadicResult := UnionUint64(items...)
	want := []uint64{3, 4, 5, 6, 7, 8, 9, 10, 11}
	if !reflect.DeepEqual(want, variadicResult) {
		t.Errorf("Received %v, wanted %v", variadicResult, want)
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
	items := [][]int64{
		{3, 4, 5},
		{6, 7, 8},
		{9, 10, 11},
	}
	variadicResult := UnionInt64(items...)
	want := []int64{3, 4, 5, 6, 7, 8, 9, 10, 11}
	if !reflect.DeepEqual(want, variadicResult) {
		t.Errorf("Received %v, wanted %v", variadicResult, want)
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

func TestUnionByteSlices(t *testing.T) {
	testCases := []struct {
		setA [][]byte
		setB [][]byte
		out  [][]byte
	}{
		{
			[][]byte{[]byte("hello"), []byte("world")},
			[][]byte{[]byte("world"), {}},
			[][]byte{[]byte("hello"), []byte("world"), {}},
		},
		{
			[][]byte{[]byte("hello")},
			[][]byte{[]byte("hello")},
			[][]byte{[]byte("hello")},
		},
	}
	for _, tt := range testCases {
		result := UnionByteSlices(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
	items := [][][]byte{
		{
			{1, 2, 3},
		},
		{
			{4, 5, 6},
		},
		{
			{7, 8, 9},
		},
	}
	variadicResult := UnionByteSlices(items...)
	want := [][]byte{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	if !reflect.DeepEqual(want, variadicResult) {
		t.Errorf("Received %v, wanted %v", variadicResult, want)
	}
}

func TestIntersectionByteSlices(t *testing.T) {
	testCases := []struct {
		input  [][][]byte
		result [][]byte
	}{
		{
			input: [][][]byte{
				{
					{1, 2, 3},
					{4, 5},
				},
				{
					{1, 2},
					{4, 5},
				},
			},
			result: [][]byte{{4, 5}},
		},
		// Ensure duplicate elements are removed in the resulting set.
		{
			input: [][][]byte{
				{
					{1, 2, 3},
					{4, 5},
					{4, 5},
				},
				{
					{1, 2},
					{4, 5},
					{4, 5},
				},
			},
			result: [][]byte{{4, 5}},
		},
		// Ensure no intersection returns an empty set.
		{
			input: [][][]byte{
				{
					{1, 2, 3},
					{4, 5},
				},
				{
					{1, 2},
				},
			},
			result: [][]byte{},
		},
		//  Intersection between A and A should return A.
		{
			input: [][][]byte{
				{
					{1, 2},
				},
				{
					{1, 2},
				},
			},
			result: [][]byte{{1, 2}},
		},
	}
	for _, tt := range testCases {
		result := IntersectionByteSlices(tt.input...)
		if !reflect.DeepEqual(result, tt.result) {
			t.Errorf("IntersectionByteSlices(%v)=%v, wanted: %v",
				tt.input, result, tt.result)
		}
	}
}

func TestInsertSort(t *testing.T) {
	testCases := []struct {
		slice     []uint64
		itemToAdd uint64
		result    []uint64
	}{
		{
			slice:     []uint64{2, 3, 5, 6, 7},
			itemToAdd: 4,
			result:    []uint64{2, 3, 4, 5, 6, 7},
		},
		{
			slice:     []uint64{2, 3, 5, 6, 7},
			itemToAdd: 1,
			result:    []uint64{1, 2, 3, 5, 6, 7},
		},
		{
			slice:     []uint64{2, 3, 5, 6, 7},
			itemToAdd: 8,
			result:    []uint64{2, 3, 5, 6, 7, 8},
		},
		{
			slice:     []uint64{2, 3, 5, 6, 7},
			itemToAdd: 0,
			result:    []uint64{0, 2, 3, 5, 6, 7},
		},
		{
			slice:     []uint64{2, 3, 5, 6, 7},
			itemToAdd: 11,
			result:    []uint64{2, 3, 5, 6, 7, 11},
		},
	}
	for _, tt := range testCases {
		result := InsertSort(tt.slice, tt.itemToAdd)
		if !reflect.DeepEqual(result, tt.result) {
			t.Errorf("InsertSort(%v,%v)=%v, wanted: %v",
				tt.slice, tt.itemToAdd, result, tt.result)
		}
	}
}

func TestTruncateSlice(t *testing.T) {
	testCases := []struct {
		slice        []uint64
		minItemVal   uint64
		result       []uint64
		truncate     bool
		itemsToTrunc []uint64
	}{
		{
			slice:        []uint64{2, 3, 5, 6, 7},
			minItemVal:   4,
			result:       []uint64{5, 6, 7},
			truncate:     true,
			itemsToTrunc: []uint64{2, 3},
		},
		{
			slice:        []uint64{2, 3, 5, 6, 7},
			minItemVal:   1,
			result:       []uint64{2, 3, 5, 6, 7},
			truncate:     false,
			itemsToTrunc: []uint64{},
		},
		{
			slice:        []uint64{2, 3, 5, 6, 7},
			minItemVal:   8,
			result:       []uint64{},
			truncate:     true,
			itemsToTrunc: []uint64{2, 3, 5, 6, 7},
		},
		{
			slice:        []uint64{2, 3, 5, 6, 7},
			minItemVal:   0,
			result:       []uint64{2, 3, 5, 6, 7},
			truncate:     false,
			itemsToTrunc: []uint64{},
		},
		{
			slice:        []uint64{2, 3, 5, 6, 7},
			minItemVal:   11,
			result:       []uint64{},
			truncate:     true,
			itemsToTrunc: []uint64{2, 3, 5, 6, 7},
		},
	}
	for _, tt := range testCases {
		truncate, result, itemsToTrunc := TruncateItems(tt.slice, tt.minItemVal)
		if truncate != tt.truncate {
			t.Fatalf("InsertSort(%v,%v) truncate=%v wanted truncate=%v",
				tt.slice, tt.minItemVal, truncate, tt.truncate)
		}
		if !reflect.DeepEqual(result, tt.result) {
			t.Errorf("InsertSort(%v,%v)=%v, wanted: %v",
				tt.slice, tt.minItemVal, result, tt.result)
		}
		if !reflect.DeepEqual(result, tt.result) {
			t.Errorf("InsertSort(%v,%v) itemsToTrunc=%v, wanted: %v",
				tt.slice, tt.minItemVal, itemsToTrunc, tt.itemsToTrunc)
		}
	}
}
