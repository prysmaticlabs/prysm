package fuzz

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/fuzz/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BeaconFuzzAttesterSlashing implements libfuzzer and beacon fuzz interface.
func BeaconFuzzAttesterSlashing(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputAttesterSlashingWrapper{}
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
	post, err := blocks.ProcessAttesterSlashings(context.Background(), st, &ethpb.BeaconBlockBody{AttesterSlashings: []*ethpb.AttesterSlashing{input.AttesterSlashing}})
	if err != nil {
		return fail(err)
	}
	return success(post)
}
