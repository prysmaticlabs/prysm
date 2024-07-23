package eth_test

import (
	"fmt"
	"reflect"
	"testing"

	fuzz "github.com/google/gofuzz"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func isZeroValue(field reflect.Value) bool {
	return reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface())
}

func fuzzCopies[T any, C eth.Copier[T]](t *testing.T, obj C) {
	fuzzer := fuzz.NewWithSeed(0)
	amount := 1000
	t.Run(fmt.Sprintf("%T", obj), func(t *testing.T) {
		for i := 0; i < amount; i++ {
			fuzzer.Fuzz(obj) // Populate thing with random values
			fuzzer.Fuzz(obj)

			got := obj.Copy()
			require.DeepEqual(t, obj, got)
			// check shallow copy working
			fuzzer.Fuzz(got)
			if !isZeroValue(reflect.ValueOf(got)) {
				require.NotEqual(t, obj, got)
			}
		}
	})
}
