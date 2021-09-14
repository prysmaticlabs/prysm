package types

import (
	"github.com/prysmaticlabs/prysm/encoding/bytes"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/metadata"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
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
	BlockMap map[[4]byte]func() (block.SignedBeaconBlock, error)
	// StateMap maps the fork-version to the underlying data type for that
	// particular fork period.
	StateMap map[[4]byte]proto.Message
	// MetaDataMap maps the fork-version to the underlying data type for that
	// particular fork period.
	MetaDataMap map[[4]byte]func() metadata.Metadata
)

// InitializeDataMaps initializes all the relevant object maps. This function is called to
// reset maps and reinitialize them.
func InitializeDataMaps() {
	// Reset our block map.
	BlockMap = map[[4]byte]func() (block.SignedBeaconBlock, error){
		bytes.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() (block.SignedBeaconBlock, error) {
			return wrapper.WrappedPhase0SignedBeaconBlock(&eth.SignedBeaconBlock{}), nil
		},
		bytes.ToBytes4(params.BeaconConfig().AltairForkVersion): func() (block.SignedBeaconBlock, error) {
			return wrapper.WrappedAltairSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}})
		},
	}

	// Reset our state map.
	StateMap = map[[4]byte]proto.Message{
		bytes.ToBytes4(params.BeaconConfig().GenesisForkVersion): &ethpb.BeaconState{},
		bytes.ToBytes4(params.BeaconConfig().AltairForkVersion):  &ethpb.BeaconStateAltair{},
	}

	// Reset our metadata map.
	MetaDataMap = map[[4]byte]func() metadata.Metadata{
		bytes.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() metadata.Metadata {
			return wrapper.WrappedMetadataV0(&ethpb.MetaDataV0{})
		},
		bytes.ToBytes4(params.BeaconConfig().AltairForkVersion): func() metadata.Metadata {
			return wrapper.WrappedMetadataV1(&ethpb.MetaDataV1{})
		},
	}
}
