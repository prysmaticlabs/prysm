package fuzz

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BeaconFuzzBlock -- TODO.
func BeaconFuzzBlock(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputBlockWithPrestate{}
	if err := input.UnmarshalSSZ(b); err != nil {
		return fail(err)
	}
	st, err := stateTrie.InitializeFromProtoUnsafe(input.State)
	if err != nil {
		return fail(err)
	}
	_, post, err := state.ProcessBlockNoVerifyAnySig(context.Background(), st, input.Block)
	if err != nil {
		return fail(err)
	}
	return success(post)
}
