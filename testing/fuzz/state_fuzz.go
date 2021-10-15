package fuzz

import (
	"context"
	"fmt"

	stateutil "github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func init() {
	features.Init(&features.Flags{
		EnableSSZCache: false,
	})
}

// BeaconStateFuzz --
func BeaconStateFuzz(input []byte) {
	params.UseMainnetConfig()
	st := &ethpb.BeaconState{}
	if err := st.UnmarshalSSZ(input); err != nil {
		return
	}
	s, err := v1.InitializeFromProtoUnsafe(st)
	if err != nil {
		panic(err)
	}
	validateStateHTR(s)
	nextEpoch := slots.ToEpoch(s.Slot()) + 1
	slot, err := slots.EpochStart(nextEpoch)
	if err != nil {
		return
	}
	if _, err := stateutil.ProcessSlots(context.Background(), s, slot); err != nil {
		_ = err
		return
	}
	validateStateHTR(s)
}

func validateStateHTR(s *v1.BeaconState) {
	rawState, ok := s.InnerStateUnsafe().(*ethpb.BeaconState)
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
