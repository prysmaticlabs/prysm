package fuzz

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateutil "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func init() {
	featureconfig.Init(&featureconfig.Flags{
		EnableSSZCache: false,
	})
}

// BeaconStateFuzz --
func BeaconStateFuzz(input []byte)  {
	st := &pb.BeaconState{}
	if err := st.UnmarshalSSZ(input); err != nil {
		return
	}
	s, err := state.InitializeFromProtoUnsafe(st)
	if err != nil {
		panic(err)
	}
	if _, err := s.HashTreeRoot(context.Background()); err != nil {
		_ = err
		return
	}
	nextEpoch := helpers.SlotToEpoch(s.Slot())+1
	if _, err := stateutil.ProcessSlots(context.Background(), s, helpers.StartSlot(nextEpoch)); err != nil {
		_ = err
		return
	}
}