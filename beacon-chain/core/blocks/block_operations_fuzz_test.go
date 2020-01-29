package blocks_test

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestFuzzProcessAttestation_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	ctx := context.Background()
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	att := &eth.Attestation{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(att)
		s, _ := beaconstate.InitializeFromProtoUnsafe(state)
		_, _ = blocks.ProcessAttestationNoVerify(ctx, s, att)
	}
}

func TestFuzzProcessBlockHeader_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	block := &eth.SignedBeaconBlock{}

	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(block)

		s, _ := beaconstate.InitializeFromProtoUnsafe(state)
		_, _ = blocks.ProcessBlockHeader(s, block)
	}
}
