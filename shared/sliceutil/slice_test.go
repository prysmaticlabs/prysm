package sliceutil

import (
	"encoding/hex"
	"fmt"
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
	col := [][][]byte{
		{
			decodeOrDie("dbe20d8d8b6a4ea77d6d827ad0b4438f26247cef7d7650ef7bd20adf4f4e31ce"),
		},
		{
			decodeOrDie("a043092233b263996ca35de19b84c46941900817da147126df2a3dc990ef18f5"),
		},
		{
			decodeOrDie("44210cf26e005ad5758e067ed5e785502bb5ed54a292572cce0746d2663d4a20"),
		},
		{
			decodeOrDie("2c5694ac395889f27f737a91dafea86f3747a2308fa2fc8c9a5c5a252659f344"),
			decodeOrDie("dbe20d8d8b6a4ea77d6d827ad0b4438f26247cef7d7650ef7bd20adf4f4e31ce"),
		},
	}
	fmt.Println(IntersectionByteSlices(col...))
}

func decodeOrDie(s string) []byte {
	res, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return res
}
