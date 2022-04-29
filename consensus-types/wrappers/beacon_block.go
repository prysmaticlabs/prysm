package wrappers

import (
	"github.com/pkg/errors"
	typeerrors "github.com/prysmaticlabs/prysm/consensus-types/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/altair"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/bellatrix"
	"github.com/prysmaticlabs/prysm/consensus-types/forks/phase0"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// WrappedSignedBeaconBlock will wrap a signed beacon block to conform to the
// signed beacon block interface.
func WrappedSignedBeaconBlock(i interface{}) (interfaces.SignedBeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return phase0.WrappedSignedBeaconBlock(b.Phase0), nil
	case *eth.SignedBeaconBlock:
		return phase0.WrappedSignedBeaconBlock(b), nil
	case *eth.GenericSignedBeaconBlock_Altair:
		return altair.WrappedSignedBeaconBlock(b.Altair)
	case *eth.SignedBeaconBlockAltair:
		return altair.WrappedSignedBeaconBlock(b)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return bellatrix.WrappedSignedBeaconBlock(b.Bellatrix)
	case *eth.SignedBeaconBlockBellatrix:
		return bellatrix.WrappedSignedBeaconBlock(b)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return bellatrix.WrappedSignedBlindedBeaconBlock(b.BlindedBellatrix)
	case *eth.SignedBlindedBeaconBlockBellatrix:
		return bellatrix.WrappedSignedBlindedBeaconBlock(b)
	case nil:
		return nil, typeerrors.ErrNilObjectWrapped
	default:
		return nil, errors.Wrapf(typeerrors.ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// WrappedBeaconBlock will wrap a signed beacon block to conform to the
// signed beacon block interface.
func WrappedBeaconBlock(i interface{}) (interfaces.BeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericBeaconBlock_Phase0:
		return phase0.WrappedBeaconBlock(b.Phase0), nil
	case *eth.BeaconBlock:
		return phase0.WrappedBeaconBlock(b), nil
	case *eth.GenericBeaconBlock_Altair:
		return altair.WrappedBeaconBlock(b.Altair)
	case *eth.BeaconBlockAltair:
		return altair.WrappedBeaconBlock(b)
	case *eth.GenericBeaconBlock_Bellatrix:
		return bellatrix.WrappedBeaconBlock(b.Bellatrix)
	case *eth.BeaconBlockBellatrix:
		return bellatrix.WrappedBeaconBlock(b)
	case *eth.GenericBeaconBlock_BlindedBellatrix:
		return bellatrix.WrappedBlindedBeaconBlock(b.BlindedBellatrix)
	case *eth.BlindedBeaconBlockBellatrix:
		return bellatrix.WrappedBlindedBeaconBlock(b)
	default:
		return nil, errors.Wrapf(typeerrors.ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// BuildSignedBeaconBlock assembles a block.SignedBeaconBlock interface compatible struct from a
// given beacon block an the appropriate signature. This method may be used to easily create a
// signed beacon block.
func BuildSignedBeaconBlock(blk interfaces.BeaconBlock, signature []byte) (interfaces.SignedBeaconBlock, error) {
	switch b := blk.(type) {
	case phase0.BeaconBlock:
		pb, ok := b.Proto().(*eth.BeaconBlock)
		if !ok {
			return nil, errors.New("unable to access inner phase0 proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlock{Block: pb, Signature: signature})
	case altair.BeaconBlock:
		pb, ok := b.Proto().(*eth.BeaconBlockAltair)
		if !ok {
			return nil, errors.New("unable to access inner altair proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockAltair{Block: pb, Signature: signature})
	case bellatrix.BeaconBlock:
		pb, ok := b.Proto().(*eth.BeaconBlockBellatrix)
		if !ok {
			return nil, errors.New("unable to access inner bellatrix proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockBellatrix{Block: pb, Signature: signature})
	case bellatrix.BlindedBeaconBlock:
		pb, ok := b.Proto().(*eth.BlindedBeaconBlockBellatrix)
		if !ok {
			return nil, errors.New("unable to access inner bellatrix proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBlindedBeaconBlockBellatrix{Block: pb, Signature: signature})
	default:
		return nil, errors.Wrapf(typeerrors.ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", b)
	}
}

func UnwrapGenericSignedBeaconBlock(gb *eth.GenericSignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
	if gb == nil {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	switch bb := gb.Block.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return WrappedSignedBeaconBlock(bb.Phase0)
	case *eth.GenericSignedBeaconBlock_Altair:
		return WrappedSignedBeaconBlock(bb.Altair)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return WrappedSignedBeaconBlock(bb.Bellatrix)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return WrappedSignedBeaconBlock(bb.BlindedBellatrix)
	default:
		return nil, errors.Wrapf(typeerrors.ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", gb)
	}
}
