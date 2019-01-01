package slices

import (
	"reflect"
	"testing"
)

func GenericTestIntersection(t *testing.T) {
	testCases := []struct {
		setA []GenericItem
		setB []GenericItem
		out  []GenericItem
	}{
		{[]GenericItem{2, 3, 5}, []GenericItem{3}, []GenericItem{3}},
		{[]GenericItem{2, 3, 5}, []GenericItem{3, 5}, []GenericItem{3, 5}},
		{[]GenericItem{2, 3, 5}, []GenericItem{5, 3, 2}, []GenericItem{5, 3, 2}},
		{[]GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}},
		{[]GenericItem{2, 3, 5}, []GenericItem{}, []GenericItem{}},
		{[]GenericItem{}, []GenericItem{2, 3, 5}, []GenericItem{}},
		{[]GenericItem{}, []GenericItem{}, []GenericItem{}},
		{[]GenericItem{1}, []GenericItem{1}, []GenericItem{1}},
	}
	for _, tt := range testCases {
		result := GenericIntersection(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func GenericTestUnion(t *testing.T) {
	testCases := []struct {
		setA []GenericItem
		setB []GenericItem
		out  []GenericItem
	}{
		{[]GenericItem{2, 3, 5}, []GenericItem{4, 6}, []GenericItem{2, 3, 5, 4, 6}},
		{[]GenericItem{2, 3, 5}, []GenericItem{3, 5}, []GenericItem{2, 3, 5}},
		{[]GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}},
		{[]GenericItem{2, 3, 5}, []GenericItem{}, []GenericItem{2, 3, 5}},
		{[]GenericItem{}, []GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}},
		{[]GenericItem{}, []GenericItem{}, []GenericItem{}},
		{[]GenericItem{1}, []GenericItem{1}, []GenericItem{1}},
	}
	for _, tt := range testCases {
		result := GenericUnion(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func GenericTestNot(t *testing.T) {
	testCases := []struct {
		setA []GenericItem
		setB []GenericItem
		out  []GenericItem
	}{
		{[]GenericItem{4, 6}, []GenericItem{2, 3, 5, 4, 6}, []GenericItem{2, 3, 5}},
		{[]GenericItem{3, 5}, []GenericItem{2, 3, 5}, []GenericItem{2}},
		{[]GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}, []GenericItem{}},
		{[]GenericItem{2}, []GenericItem{2, 3, 5}, []GenericItem{3, 5}},
		{[]GenericItem{}, []GenericItem{2, 3, 5}, []GenericItem{2, 3, 5}},
		{[]GenericItem{}, []GenericItem{}, []GenericItem{}},
		{[]GenericItem{1}, []GenericItem{1}, []GenericItem{}},
	}
	for _, tt := range testCases {
		result := GenericNot(tt.setA, tt.setB)
		if !reflect.DeepEqual(result, tt.out) {
			t.Errorf("got %d, want %d", result, tt.out)
		}
	}
}

func GenericTestIsIn(t *testing.T) {
	testCases := []struct {
		a      GenericItem
		b      []GenericItem
		result bool
	}{
		{0, []GenericItem{}, false},
		{0, []GenericItem{0}, true},
		{4, []GenericItem{2, 3, 5, 4, 6}, true},
		{100, []GenericItem{2, 3, 5, 4, 6}, false},
	}
	for _, tt := range testCases {
		result := GenericIsIn(tt.a, tt.b)
		if result != tt.result {
			t.Errorf("IsIn(%d, %v)=%v, wanted: %v",
				tt.a, tt.b, result, tt.result)
		}
	}
}
