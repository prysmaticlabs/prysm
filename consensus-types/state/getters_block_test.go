package state

import (
	"testing"

	testtmpl "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/testing"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func TestBeaconState_LatestBlockHeader_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{})
		},
		func(BH *ethpb.BeaconBlockHeader) (types.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{LatestBlockHeader: BH})
		},
	)
}

func TestBeaconState_LatestBlockHeader_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		},
		func(BH *ethpb.BeaconBlockHeader) (types.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{LatestBlockHeader: BH})
		},
	)
}

func TestBeaconState_LatestBlockHeader_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{})
		},
		func(BH *ethpb.BeaconBlockHeader) (types.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{LatestBlockHeader: BH})
		},
	)
}

func TestBeaconState_LatestBlockHeader_Capella(t *testing.T) {
	testtmpl.VerifyBeaconStateLatestBlockHeader(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoCapella(&ethpb.BeaconStateCapella{})
		},
		func(BH *ethpb.BeaconBlockHeader) (types.BeaconState, error) {
			return InitializeFromProtoCapella(&ethpb.BeaconStateCapella{LatestBlockHeader: BH})
		},
	)
}

func TestBeaconState_BlockRoots_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRoots_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRoots_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRoots_Capella(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootsNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoCapella(&ethpb.BeaconStateCapella{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoCapella(&ethpb.BeaconStateCapella{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Phase0(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoPhase0(&ethpb.BeaconState{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Altair(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoAltair(&ethpb.BeaconStateAltair{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Bellatrix(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoBellatrix(&ethpb.BeaconStateBellatrix{BlockRoots: BR})
		},
	)
}

func TestBeaconState_BlockRootAtIndex_Capella(t *testing.T) {
	testtmpl.VerifyBeaconStateBlockRootAtIndexNative(
		t,
		func() (types.BeaconState, error) {
			return InitializeFromProtoCapella(&ethpb.BeaconStateCapella{})
		},
		func(BR [][]byte) (types.BeaconState, error) {
			return InitializeFromProtoCapella(&ethpb.BeaconStateCapella{BlockRoots: BR})
		},
	)
}
