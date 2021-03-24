package fuzz

import (
	"context"

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
	nextEpoch := helpers.SlotToEpoch(s.Slot()) + 1
	slot, err := helpers.StartSlot(nextEpoch)
	if err != nil {
		return
	}
	if _, err := stateutil.ProcessSlots(context.Background(), s, slot); err != nil {
		_ = err
		return
	}
}
