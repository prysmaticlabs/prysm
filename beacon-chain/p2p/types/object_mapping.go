package types

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/protobuf/proto"
)

func init() {
	// Initialize data maps.
	InitializeDataMaps()
}

// This file provides a mapping of fork version to the respective data type. This is
// to allow any service to appropriately use the correct data type with a provided
// fork-version.

var (
	// BlockMap maps the fork-version to the underlying data type for that
	// particular fork period.
	BlockMap map[[4]byte]func() interfaces.SignedBeaconBlock
	// StateMap maps the fork-version to the underlying data type for that
	// particular fork period.
	StateMap map[[4]byte]proto.Message
	// MetaDataMap maps the fork-version to the underlying data type for that
	// particular fork period.
	MetaDataMap map[[4]byte]func() interfaces.Metadata
)

// InitializeDataMaps initializes all the relevant object maps. This function is called to
// reset maps and reinitialize them.
func InitializeDataMaps() {
	// Reset our block map.
	BlockMap = map[[4]byte]func() interfaces.SignedBeaconBlock{
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() interfaces.SignedBeaconBlock {
			return interfaces.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{})
		},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() interfaces.SignedBeaconBlock {
			return interfaces.WrappedAltairSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{})
		},
	}

	// Reset our state map.
	StateMap = map[[4]byte]proto.Message{
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): &pbp2p.BeaconState{},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion):  &pbp2p.BeaconStateAltair{},
	}

	// Reset our metadata map.
	MetaDataMap = map[[4]byte]func() interfaces.Metadata{
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() interfaces.Metadata {
			return interfaces.WrappedMetadataV0(&pbp2p.MetaDataV0{})
		},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() interfaces.Metadata {
			return interfaces.WrappedMetadataV1(&pbp2p.MetaDataV1{})
		},
	}
}
