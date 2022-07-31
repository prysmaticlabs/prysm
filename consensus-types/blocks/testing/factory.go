package testing

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/blocks"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// NewSignedBeaconBlockFromGeneric creates a signed beacon block
// from a protobuf generic signed beacon block.
func NewSignedBeaconBlockFromGeneric(gb *eth.GenericSignedBeaconBlock) (*blocks.SignedBeaconBlock, error) {
	if gb == nil {
		return nil, blocks.ErrNilObjectWrapped
	}
	switch bb := gb.Block.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return blocks.NewSignedBeaconBlock(bb.Phase0)
	case *eth.GenericSignedBeaconBlock_Altair:
		return blocks.NewSignedBeaconBlock(bb.Altair)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return blocks.NewSignedBeaconBlock(bb.Bellatrix)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return blocks.NewSignedBeaconBlock(bb.BlindedBellatrix)
	default:
		return nil, errors.Wrapf(blocks.ErrUnsupportedSignedBeaconBlock, "unable to create block from type %T", gb)
	}
}
