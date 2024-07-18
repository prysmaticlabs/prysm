package enginev1_test

import (
	"fmt"
	"testing"

	fuzz "github.com/google/gofuzz"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestCopyExecutionPayloadElectra_1000(t *testing.T) {
	payload := &enginev1.ExecutionPayloadElectra{}
	fuzzer := fuzz.NewWithSeed(0)
	t.Run(fmt.Sprintf("%T", payload), func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			fuzzer.Fuzz(payload) // Populate thing with random values
			got := payload.Copy()
			require.DeepEqual(t, payload, got)
		}
	})
}
