package v3

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/beacon-chain/state/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func TestBeaconState_SlotDataRace(t *testing.T) {
	testtmpl.VerifyBeaconState_SlotDataRace(t, func() (state.BeaconState, error) {
		return InitializeFromProto(&ethpb.BeaconStateBellatrix{Slot: 1})
	})
}

func TestBeaconState_MatchCurrentJustifiedCheckpt(t *testing.T) {
	testtmpl.VerifyBeaconState_MatchCurrentJustifiedCheckptNative(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconStateBellatrix{CurrentJustifiedCheckpoint: cp})
		},
	)
}

func TestBeaconState_MatchPreviousJustifiedCheckpt(t *testing.T) {
	testtmpl.VerifyBeaconState_MatchPreviousJustifiedCheckptNative(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconStateBellatrix{PreviousJustifiedCheckpoint: cp})
		},
	)
}
