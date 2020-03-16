package fuzz

import (
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/beacon-chain/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func BeaconFuzzDeposit(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputDepositWrapper{}
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
	post, err := blocks.ProcessDeposit(st, input.Deposit)
	if err != nil {
		return fail(err)
	}

	result, err := ssz.Marshal(post.InnerStateUnsafe())
	if err != nil {
		panic(err)
	}
	return result, true
}
