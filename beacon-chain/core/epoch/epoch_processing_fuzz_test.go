package epoch

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestFuzzFinalUpdates_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	base := &ethpb.BeaconState{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(base)
		s, err := v1.InitializeFromProtoUnsafe(base)
		require.NoError(t, err)
		_, err = ProcessFinalUpdates(s)
		_ = err
	}
}
