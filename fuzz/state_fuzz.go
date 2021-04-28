package fuzz

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateutil "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.Init(&featureconfig.Flags{
		EnableSSZCache: false,
	})
}

// FuzzState wraps BeaconStateFuzz in a go-fuzz compatible interface
func FuzzState(b []byte) int {
	BeaconStateFuzz(b)
	return 0
}

// BeaconStateFuzz --
func BeaconStateFuzz(input []byte) {
	params.UseMainnetConfig()
	st := &pb.BeaconState{}
	if err := st.UnmarshalSSZ(input); err != nil {
		return
	}
	s, err := stateV0.InitializeFromProtoUnsafe(st)
	if err != nil {
		panic(err)
	}
	validateStateHTR(s)
	nextEpoch := helpers.SlotToEpoch(s.Slot()) + 1
	slot, err := helpers.StartSlot(nextEpoch)
	if err != nil {
		return
	}
	if _, err := stateutil.ProcessSlots(context.Background(), s, slot); err != nil {
		_ = err
		return
	}
	validateStateHTR(s)
}

func validateStateHTR(s *stateV0.BeaconState) {
	rawState, ok := s.InnerStateUnsafe().(*pb.BeaconState)
	if !ok {
		panic("non valid type assertion")
	}
	rt, err := s.HashTreeRoot(context.Background())
	nxtRt, err2 := rawState.HashTreeRoot()

	if err == nil && err2 != nil {
		panic("HTR from state had only and error from cached state HTR method")
	}
	if err != nil && err2 == nil {
		panic("HTR from state had only and error from fast-ssz HTR method")
	}
	if err != nil && err2 != nil {
		return
	}
	if rt != nxtRt {
		panic(fmt.Sprintf("cached HTR gave a root of %#x while fast-ssz gave a root of %#x", rt, nxtRt))
	}
}
