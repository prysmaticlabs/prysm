package epoch

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestFuzzFinalUpdates_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	base := &ethereum_beacon_p2p_v1.BeaconState{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(base)
		s, err := beaconstate.InitializeFromProtoUnsafe(base)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = ProcessFinalUpdates(s)
	}
}
