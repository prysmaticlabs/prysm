package fuzz

import (
	"context"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/beacon-chain/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func BeaconFuzzAttestation(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputAttestationWrapper{}
	if err := ssz.Unmarshal(b, input); err != nil {
		return fail(err)
	}
	s := prylabs_testing.GetBeaconFuzzState(input.StateID)
	if s == nil {
		return nil, false
	}
	st, err := stateTrie.InitializeFromProto(s)
	if err != nil {
		return fail(err)
	}
	if input.Attestation == nil || input.Attestation.Data == nil {
		return nil, false
	}
	st, err = state.ProcessSlots(context.Background(), st, input.Attestation.Data.Slot)
	if err != nil {
		return fail(err)
	}
	post, err := blocks.ProcessAttestationNoVerify(context.Background(), st, input.Attestation)
	if err != nil {
		return fail(err)
	}

	result, err := ssz.Marshal(post.InnerStateUnsafe())
	if err != nil {
		panic(err)
	}
	return result, true
}
