package v1

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/testing"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func TestBeaconState_JustificationBitsNil(t *testing.T) {
	testtmpl.VerifyBeaconStateJustificationBitsNil(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{})
		})
}

func TestBeaconState_JustificationBits(t *testing.T) {
	testtmpl.VerifyBeaconStateJustificationBits(
		t,
		func(bits bitfield.Bitvector4) (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{JustificationBits: bits})
		})
}

func TestBeaconState_PreviousJustifiedCheckpointNil(t *testing.T) {
	testtmpl.VerifyBeaconStatePreviousJustifiedCheckpointNil(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{})
		})
}

func TestBeaconState_PreviousJustifiedCheckpoint(t *testing.T) {
	testtmpl.VerifyBeaconStatePreviousJustifiedCheckpoint(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{PreviousJustifiedCheckpoint: cp})
		})
}

func TestBeaconState_CurrentJustifiedCheckpointNil(t *testing.T) {
	testtmpl.VerifyBeaconStateCurrentJustifiedCheckpointNil(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{})
		})
}

func TestBeaconState_CurrentJustifiedCheckpoint(t *testing.T) {
	testtmpl.VerifyBeaconStateCurrentJustifiedCheckpoint(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{CurrentJustifiedCheckpoint: cp})
		})
}

func TestBeaconState_FinalizedCheckpointNil(t *testing.T) {
	testtmpl.VerifyBeaconStateFinalizedCheckpointNil(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{})
		})
}

func TestBeaconState_FinalizedCheckpoint(t *testing.T) {
	testtmpl.VerifyBeaconStateFinalizedCheckpoint(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProtoUnsafe(&ethpb.BeaconState{FinalizedCheckpoint: cp})
		})
}
