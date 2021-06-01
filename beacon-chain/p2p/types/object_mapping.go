package types

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/protobuf/proto"
)

// This file provides a mapping of fork version to the respective data type. This is
// to allow any service to appropriately use the correct data type with a provided
// fork-version.

// BlockMap maps the fork-version to the underlying data type for that
// particular fork period.
var BlockMap = map[[4]byte]func() interfaces.SignedBeaconBlock{
	bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() interfaces.SignedBeaconBlock {
		return interfaces.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{})
	},
	bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() interfaces.SignedBeaconBlock {
		return interfaces.WrappedAltairSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{})
	},
}

// StateMap maps the fork-version to the underlying data type for that
// particular fork period.
var StateMap = map[[4]byte]proto.Message{
	bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): &pbp2p.BeaconState{},
	bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion):  &pbp2p.BeaconStateAltair{},
}

// MetaDataMap maps the fork-version to the underlying data type for that
// particular fork period.
var MetaDataMap = map[[4]byte]func() proto.Message{
	bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() proto.Message {
		return &pbp2p.MetaData{}
	},
	bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() proto.Message {
		return &pbp2p.MetaDataV2{}
	},
}
