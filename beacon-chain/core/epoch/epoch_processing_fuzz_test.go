package epoch

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestFuzzFinalUpdates_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	base := &ethpb.BeaconState{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(base)
		s, err := state.InitializeFromProtoUnsafePhase0(base)
		require.NoError(t, err)
		_, err = ProcessFinalUpdates(s)
		_ = err
	}
}
