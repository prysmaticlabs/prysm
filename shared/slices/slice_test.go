package slices

import (
	"reflect"
	"testing"
)

func TestIntersection(t *testing.T) {
	testCases := []struct {
		setA []uint32
		setB []uint32
		out  []uint32
	}{
		{[]uint32{2, 3, 5}, []uint32{3}, []uint32{3}},
		{[]uint32{2, 3, 5}, []uint32{3, 5}, []uint32{3, 5}},
		{[]uint32{2, 3, 5}, []uint32{5, 3, 2}, []uint32{5, 3, 2}},
		{[]uint32{2, 3, 5}, []uint32{2, 3, 5}, []uint32{2, 3, 5}},
		{[]uint32{2, 3, 5}, []uint32{}, []uint32{}},
		{[]uint32{}, []uint32{2, 3, 5}, []uint32{}},
		{[]uint32{}, []uint32{}, []uint32{}},
		{[]uint32{1}, []uint32{1}, []uint32{1}},
	}
	for _, tt := range testCases {
		result := Intersection(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func TestUnion(t *testing.T) {
	testCases := []struct {
		setA []uint32
		setB []uint32
		out  []uint32
	}{
		{[]uint32{2, 3, 5}, []uint32{4, 6}, []uint32{2, 3, 5, 4, 6}},
		{[]uint32{2, 3, 5}, []uint32{3, 5}, []uint32{2, 3, 5}},
		{[]uint32{2, 3, 5}, []uint32{2, 3, 5}, []uint32{2, 3, 5}},
		{[]uint32{2, 3, 5}, []uint32{}, []uint32{2, 3, 5}},
		{[]uint32{}, []uint32{2, 3, 5}, []uint32{2, 3, 5}},
		{[]uint32{}, []uint32{}, []uint32{}},
		{[]uint32{1}, []uint32{1}, []uint32{1}},
	}
	for _, tt := range testCases {
		result := Union(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}
