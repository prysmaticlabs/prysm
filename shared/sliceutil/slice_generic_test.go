package sliceutil

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func TestGenericIntersection(t *testing.T) {
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
		result, err := GenericIntersection(tt.setA, tt.setB)
		if err != nil {
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %d", result, tt.out)
			}
		}

	}

}

func TestGenericIntersectionWithSSZ(t *testing.T) {
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
		b1 := new(bytes.Buffer)
		err := ssz.Encode(b1, tt.setA)

		b2 := new(bytes.Buffer)
		err1 := ssz.Encode(b2, tt.setA)
		if err1 == nil && err == nil {

			result, err := GenericIntersection(b1.Bytes(), b2.Bytes())
			if err != nil {
				if !reflect.DeepEqual(result, tt.out) {
					t.Errorf("got %v, want %d", result, tt.out)
				}
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

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %v", result, tt.out)
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

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %v", result, tt.out)
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

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %d", result, tt.out)
			}
		}

	}

}

func TestGenericNot(t *testing.T) {
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
		result, err := GenericNot(tt.setA, tt.setB)
		if err != nil {

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %d", result, tt.out)
			}
		}

	}
}

func TestFloatGenericNot(t *testing.T) {
	testCases := []struct {
		setA []float32
		setB []float32
		out  []float32
	}{
		{[]float32{4, 6}, []float32{2, 3, 5, 4, 6}, []float32{2, 3, 5}},
		{[]float32{3, 5}, []float32{2, 3, 5}, []float32{2}},
		{[]float32{2, 3, 5}, []float32{2, 3, 5}, []float32{}},
		{[]float32{2}, []float32{2, 3, 5}, []float32{3, 5}},
		{[]float32{}, []float32{2, 3, 5}, []float32{2, 3, 5}},
		{[]float32{}, []float32{}, []float32{}},
		{[]float32{1}, []float32{1}, []float32{}},
	}
	for _, tt := range testCases {
		result, err := GenericNot(tt.setA, tt.setB)
		if err != nil {

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %v", result, tt.out)
			}
		}

	}
}

func TestStringGenericNot(t *testing.T) {
	testCases := []struct {
		setA []string
		setB []string
		out  []string
	}{
		{[]string{"hello", "world"}, []string{"hello", "world", "its", "go"}, []string{"its", "go"}},
		{[]string{}, []string{}, []string{}},
	}
	for _, tt := range testCases {
		result, err := GenericNot(tt.setA, tt.setB)
		if err != nil {

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %v", result, tt.out)
			}
		}

	}
}

func TestIntGenericNot(t *testing.T) {
	testCases := []struct {
		setA []int32
		setB []int32
		out  []int32
	}{
		{[]int32{4, 6}, []int32{2, 3, 5, 4, 6}, []int32{2, 3, 5}},
		{[]int32{3, 5}, []int32{2, 3, 5}, []int32{2}},
		{[]int32{2, 3, 5}, []int32{2, 3, 5}, []int32{}},
		{[]int32{2}, []int32{2, 3, 5}, []int32{3, 5}},
		{[]int32{}, []int32{2, 3, 5}, []int32{2, 3, 5}},
		{[]int32{}, []int32{}, []int32{}},
		{[]int32{1}, []int32{1}, []int32{}},
	}
	for _, tt := range testCases {
		result, err := GenericNot(tt.setA, tt.setB)
		if err != nil {

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %d", result, tt.out)
			}
		}

	}
}

func TestGenericUnion(t *testing.T) {
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
		result, err := GenericUnion(tt.setA, tt.setB)
		if err != nil {

			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %d", result, tt.out)
			}
		}

	}
}

func TestFloatGenericUnion(t *testing.T) {
	testCases := []struct {
		setA []float32
		setB []float32
		out  []float32
	}{
		{[]float32{2, 3, 5}, []float32{4, 6}, []float32{2, 3, 5, 4, 6}},
		{[]float32{2, 3, 5}, []float32{4, 6}, []float32{2, 3, 5, 4, 6}},
		{[]float32{2, 3, 5}, []float32{3, 5}, []float32{2, 3, 5}},
		{[]float32{2, 3, 5}, []float32{2, 3, 5}, []float32{2, 3, 5}},
		{[]float32{2, 3, 5}, []float32{}, []float32{2, 3, 5}},
		{[]float32{}, []float32{2, 3, 5}, []float32{2, 3, 5}},
		{[]float32{}, []float32{}, []float32{}},
		{[]float32{1}, []float32{1}, []float32{1}},
	}
	for _, tt := range testCases {
		result, err := GenericUnion(tt.setA, tt.setB)
		if err != nil {
			if !reflect.DeepEqual(result, tt.out) {
				t.Errorf("got %v, want %v", result, tt.out)
			}
		}

	}
}

func TestGenericIsIn(t *testing.T) {
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
		result := GenericIsIn(tt.a, tt.b)
		if result != tt.result {
			t.Errorf("IsIn(%d, %v)=%v, wanted: %v",
				tt.a, tt.b, result, tt.result)
		}
	}
}

func BenchmarkGenericIntersection(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			res, err := GenericIntersection(tt.setA, tt.setB)
			if err != nil {
				b.Errorf("Benchmark error for %v", res)
			}

		}

	}
}

func BenchmarkIntersection(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			IntersectionUint64(tt.setA, tt.setB)

		}
	}
}

func BenchmarkUnion(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			UnionUint64(tt.setA, tt.setB)

		}

	}
}

func BenchmarkGenericUnion(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			res, err := GenericUnion(tt.setA, tt.setB)
			if err != nil {
				b.Errorf("Benchmark error for %v", res)
			}

		}
	}
}

func BenchmarkNot(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			NotUint64(tt.setA, tt.setB)

		}

	}
}

func BenchmarkGenericNot(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			res, err := GenericNot(tt.setA, tt.setB)
			if err != nil {
				b.Errorf("Benchmark error for %v", res)
			}

		}

	}
}

func BenchmarkIsIn(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			IsInUint64(tt.a, tt.b)

		}

	}
}

func BenchmarkGenericIsIn(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			GenericIsIn(tt.a, tt.b)

		}

	}
}

func BenchmarkGenericIntersectionWithSSZ(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			b1 := new(bytes.Buffer)
			err := ssz.Encode(b1, tt.setA)

			b2 := new(bytes.Buffer)
			err1 := ssz.Encode(b2, tt.setA)
			if err1 == nil && err == nil {

				res, err := GenericIntersection(b1.Bytes(), b2.Bytes())
				if err != nil {
					b.Errorf("Benchmark error for %v", res)
				}
			}

		}
	}
}

func BenchmarkIntersectionWithSSZ(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			b1 := new(bytes.Buffer)
			err := ssz.Encode(b1, tt.setA)

			b2 := new(bytes.Buffer)
			err1 := ssz.Encode(b2, tt.setA)
			if err1 == nil && err == nil {

				ByteIntersection(b1.Bytes(), b2.Bytes())

			}

		}
	}
}

func BenchmarkGenericUnionWithSSZ(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			b1 := new(bytes.Buffer)
			err := ssz.Encode(b1, tt.setA)

			b2 := new(bytes.Buffer)
			err1 := ssz.Encode(b2, tt.setA)
			if err1 == nil && err == nil {

				res, err := GenericUnion(b1.Bytes(), b2.Bytes())
				if err != nil {
					b.Errorf("Benchmark error for %v", res)
				}
			}

		}
	}
}

func BenchmarkUnionWithSSZ(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			b1 := new(bytes.Buffer)
			err := ssz.Encode(b1, tt.setA)

			b2 := new(bytes.Buffer)
			err1 := ssz.Encode(b2, tt.setA)
			if err1 == nil && err == nil {

				ByteUnion(b1.Bytes(), b2.Bytes())

			}

		}
	}
}

func BenchmarkGenericNotWithSSZ(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			b1 := new(bytes.Buffer)
			err := ssz.Encode(b1, tt.setA)

			b2 := new(bytes.Buffer)
			err1 := ssz.Encode(b2, tt.setA)
			if err1 == nil && err == nil {

				res, err := GenericNot(b1.Bytes(), b2.Bytes())
				if err != nil {
					b.Errorf("Benchmark error for %v", res)
				}
			}

		}
	}
}

func BenchmarkNotWithSSZ(b *testing.B) {
	for i := 0; i < b.N; i++ {
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
			b1 := new(bytes.Buffer)
			err := ssz.Encode(b1, tt.setA)

			b2 := new(bytes.Buffer)
			err1 := ssz.Encode(b2, tt.setA)
			if err1 == nil && err == nil {
				ByteNot(b1.Bytes(), b2.Bytes())

			}

		}
	}
}
