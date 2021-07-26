package types

import (
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	wrapperv1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/metadata"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	wrapperv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() (block.SignedBeaconBlock, error) {
			return wrapperv1.WrappedPhase0SignedBeaconBlock(&eth.SignedBeaconBlock{}), nil
		},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() (block.SignedBeaconBlock, error) {
			return wrapperv2.WrappedAltairSignedBeaconBlock(&prysmv2.SignedBeaconBlock{Block: &prysmv2.BeaconBlockAltair{}})
		},
	}

	// Reset our state map.
	StateMap = map[[4]byte]proto.Message{
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): &statepb.BeaconState{},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion):  &statepb.BeaconStateAltair{},
	}

	// Reset our metadata map.
	MetaDataMap = map[[4]byte]func() metadata.Metadata{
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() metadata.Metadata {
			return wrapperv2.WrappedMetadataV0(&prysmv2.MetaDataV0{})
		},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() metadata.Metadata {
			return wrapperv2.WrappedMetadataV1(&prysmv2.MetaDataV1{})
		},
	}
}
