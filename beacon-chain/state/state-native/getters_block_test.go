package state_native

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/testing"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func TestBeaconState_LatestBlockHeader_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{})
		},
		func(BH *ethpb.BeaconBlockHeader) (state.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{LatestBlockHeader: BH})
		},
	)
}

func TestBeaconState_LatestBlockHeader_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		},
		func(BH *ethpb.BeaconBlockHeader) (state.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{LatestBlockHeader: BH})
		},
	)
}

func TestBeaconState_LatestBlockHeader_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{}})
		},
		func(BH *ethpb.BeaconBlockHeader) (state.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{LatestBlockHeader: BH, LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{}})
		},
	)
}

func TestBeaconState_LatestBlockHeader_4844(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto4844(&ethpb.BeaconState4844{LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader4844{}})
		},
		func(BH *ethpb.BeaconBlockHeader) (state.BeaconState, error) {
			return InitializeFromProto4844(&ethpb.BeaconState4844{LatestBlockHeader: BH, LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader4844{}})
		},
	)
}

func TestBeaconState_BlockRoots_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRoots_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRoots_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{}})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{BlockRoots: BR, LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{}})
		},
	)
}

func TestBeaconState_BlockRoots_4844(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto4844(&ethpb.BeaconState4844{LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader4844{}})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProto4844(&ethpb.BeaconState4844{BlockRoots: BR, LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader4844{}})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{}})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{BlockRoots: BR, LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{}})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_4844(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto4844(&ethpb.BeaconState4844{LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader4844{}})
		},
		func(BR [][]byte) (state.BeaconState, error) {
			return InitializeFromProto4844(&ethpb.BeaconState4844{BlockRoots: BR, LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader4844{}})
		},
	)
}
