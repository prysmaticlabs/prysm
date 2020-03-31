// +build libfuzzer

package fuzz

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/beacon-chain/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func BeaconFuzzAttesterSlashing(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputAttesterSlashingWrapper{}
	if err := input.UnmarshalSSZ(b); err != nil {
	//if err := ssz.Unmarshal(b, input); err != nil {
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
	post, err := blocks.ProcessAttesterSlashings(context.Background(), st, &ethpb.BeaconBlockBody{AttesterSlashings: []*ethpb.AttesterSlashing{input.AttesterSlashing}})
	if err != nil {
		return fail(err)
	}
	return success(post)
}
