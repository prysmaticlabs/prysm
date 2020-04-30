// +build libfuzzer

package fuzz

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/fuzz/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func BeaconFuzzVoluntaryExit(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputVoluntaryExitWrapper{}
	if err := input.UnmarshalSSZ(b); err != nil {
		//if err := ssz.Unmarshal(b, input); err != nil {
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
	post, err := blocks.ProcessVoluntaryExitsNoVerify(st, &ethpb.BeaconBlockBody{VoluntaryExits: []*ethpb.SignedVoluntaryExit{{Exit: input.VoluntaryExit}}})
	if err != nil {
		return fail(err)
	}
	return success(post)
}
