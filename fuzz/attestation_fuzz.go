package fuzz

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/fuzz/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BeaconFuzzAttestation implements libfuzzer and beacon fuzz interface.
func BeaconFuzzAttestation(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputAttestationWrapper{}
	if err := input.UnmarshalSSZ(b); err != nil {
		return fail(err)
	}
	s, err := prylabs_testing.GetBeaconFuzzState(input.StateID)
	if err != nil || s == nil {
		return nil, false
	}
	st, err := stateTrie.InitializeFromProto(s)
	if err != nil {
		return fail(err)
	}
	post, err := blocks.ProcessAttestationNoVerifySignature(context.Background(), st, input.Attestation)
	if err != nil {
		return fail(err)
	}

	return success(post)
}
