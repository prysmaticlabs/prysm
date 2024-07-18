package enginev1_test

import (
	"fmt"
	"testing"

	fuzz "github.com/google/gofuzz"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestCopyExecutionPayload_Fuzz(t *testing.T) {
	fuzzCloneable(t, &enginev1.ExecutionPayloadElectra{})
}

func fuzzCloneable[T any, C enginev1.Cloneable[T]](t *testing.T, obj C) {
	fuzzer := fuzz.NewWithSeed(0)
	amount := 1000
	t.Run(fmt.Sprintf("%T", obj), func(t *testing.T) {
		for i := 0; i < amount; i++ {
			fuzzer.Fuzz(obj) // Populate thing with random values
			got := obj.Copy()
			require.DeepEqual(t, obj, got)
		}
	})
}
