package slices

import (
	"reflect"
	"testing"
)

func TestGenericIntersection(t *testing.T) {
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
		result, err := GenericIntersection(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]uint32)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}

}

func TestFloatGenericIntersection(t *testing.T) {
	testCases := []struct {
		setA []float32
		setB []float32
		out  []float32
	}{
		{[]float32{2, 3, 5}, []float32{3}, []float32{3}},
		{[]float32{2, 3, 5}, []float32{3, 5}, []float32{3, 5}},
		{[]float32{2, 3, 5}, []float32{5, 3, 2}, []float32{5, 3, 2}},
		{[]float32{2, 3, 5}, []float32{2, 3, 5}, []float32{2, 3, 5}},
		{[]float32{2, 3, 5}, []float32{}, []float32{}},
		{[]float32{}, []float32{2, 3, 5}, []float32{}},
		{[]float32{}, []float32{}, []float32{}},
		{[]float32{1}, []float32{1}, []float32{1}},
	}
	for _, tt := range testCases {
		result, err := GenericIntersection(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]float32)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}

}

func TestStringGenericIntersection(t *testing.T) {
	testCases := []struct {
		setA []string
		setB []string
		out  []string
	}{
		{[]string{"hello", "world"}, []string{"world"}, []string{"world"}},
		{[]string{"hello"}, []string{"world"}, []string{}},
	}
	for _, tt := range testCases {
		result, err := GenericIntersection(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]string)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}

}

func TestIntGenericIntersection(t *testing.T) {
	testCases := []struct {
		setA []int32
		setB []int32
		out  []int32
	}{
		{[]int32{2, 3, 5}, []int32{3}, []int32{3}},
		{[]int32{2, 3, 5}, []int32{3, 5}, []int32{3, 5}},
		{[]int32{2, 3, 5}, []int32{5, 3, 2}, []int32{5, 3, 2}},
		{[]int32{2, 3, 5}, []int32{2, 3, 5}, []int32{2, 3, 5}},
		{[]int32{2, 3, 5}, []int32{}, []int32{}},
		{[]int32{}, []int32{2, 3, 5}, []int32{}},
		{[]int32{}, []int32{}, []int32{}},
		{[]int32{1}, []int32{1}, []int32{1}},
	}
	for _, tt := range testCases {
		result, err := GenericIntersection(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]int32)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}

}

func TestGenericNot(t *testing.T) {
	testCases := []struct {
		setA []uint32
		setB []uint32
		out  []uint32
	}{
		{[]uint32{4, 6}, []uint32{2, 3, 5, 4, 6}, []uint32{2, 3, 5}},
		{[]uint32{3, 5}, []uint32{2, 3, 5}, []uint32{2}},
		{[]uint32{2, 3, 5}, []uint32{2, 3, 5}, []uint32{}},
		{[]uint32{2}, []uint32{2, 3, 5}, []uint32{3, 5}},
		{[]uint32{}, []uint32{2, 3, 5}, []uint32{2, 3, 5}},
		{[]uint32{}, []uint32{}, []uint32{}},
		{[]uint32{1}, []uint32{1}, []uint32{}},
	}
	for _, tt := range testCases {
		result, err := GenericNot(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]uint32)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}
}

func TestFloatGenericNot(t *testing.T) {
	testCases := []struct {
		setA []uint32
		setB []uint32
		out  []uint32
	}{
		{[]uint32{4, 6}, []uint32{2, 3, 5, 4, 6}, []uint32{2, 3, 5}},
		{[]uint32{3, 5}, []uint32{2, 3, 5}, []uint32{2}},
		{[]uint32{2, 3, 5}, []uint32{2, 3, 5}, []uint32{}},
		{[]uint32{2}, []uint32{2, 3, 5}, []uint32{3, 5}},
		{[]uint32{}, []uint32{2, 3, 5}, []uint32{2, 3, 5}},
		{[]uint32{}, []uint32{}, []uint32{}},
		{[]uint32{1}, []uint32{1}, []uint32{}},
	}
	for _, tt := range testCases {
		result, err := GenericNot(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]uint32)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}
}

func TestGenericUnion(t *testing.T) {
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
		result, err := GenericUnion(tt.setA, tt.setB)
		if err != nil {
			result := result.Interface().([]uint32)
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %d, want %d", result, tt.out)
			}
		}

	}
}

func TestGenericIsIn(t *testing.T) {
	testCases := []struct {
		a      uint32
		b      []uint32
		result bool
	}{
		{0, []uint32{}, false},
		{0, []uint32{0}, true},
		{4, []uint32{2, 3, 5, 4, 6}, true},
		{100, []uint32{2, 3, 5, 4, 6}, false},
	}
	for _, tt := range testCases {
		result := GenericIsIn(tt.a, tt.b)
		if result != tt.result {
			t.Errorf("IsIn(%d, %v)=%v, wanted: %v",
				tt.a, tt.b, result, tt.result)
		}
	}
}
