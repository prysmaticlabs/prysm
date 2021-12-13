package types

import (
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/metadata"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
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
			return wrapper.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{}), nil
		},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() (block.SignedBeaconBlock, error) {
			return wrapper.WrappedAltairSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}})
		},
	}

	// Reset our metadata map.
	MetaDataMap = map[[4]byte]func() metadata.Metadata{
		bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion): func() metadata.Metadata {
			return wrapper.WrappedMetadataV0(&ethpb.MetaDataV0{})
		},
		bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion): func() metadata.Metadata {
			return wrapper.WrappedMetadataV1(&ethpb.MetaDataV1{})
		},
	}
}
