package wrapper

import (
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

var (
	// ErrUnsupportedSignedBeaconBlock is returned when the struct type is not a supported signed
	// beacon block type.
	ErrUnsupportedSignedBeaconBlock = errors.New("unsupported signed beacon block")
	// ErrUnsupportedBeaconBlock is returned when the struct type is not a supported beacon block
	// type.
	ErrUnsupportedBeaconBlock = errors.New("unsupported beacon block")
	// ErrUnsupportedPhase0Block is returned when accessing a phase0 block from a non-phase0 wrapped
	// block.
	ErrUnsupportedPhase0Block = errors.New("unsupported phase0 block")
	// ErrUnsupportedAltairBlock is returned when accessing an altair block from non-altair wrapped
	// block.
	ErrUnsupportedAltairBlock = errors.New("unsupported altair block")
	// ErrUnsupportedBellatrixBlock is returned when accessing a bellatrix block from a non-bellatrix wrapped
	// block.
	ErrUnsupportedBellatrixBlock = errors.New("unsupported bellatrix block")
	// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
	ErrNilObjectWrapped = errors.New("attempted to wrap nil object")
)

// WrappedSignedBeaconBlock will wrap a signed beacon block to conform to the
// signed beacon block interface.
func WrappedSignedBeaconBlock(i interface{}) (block.SignedBeaconBlock, error) {
	switch b := i.(type) {
	case *eth.SignedBeaconBlock:
		return WrappedPhase0SignedBeaconBlock(b), nil
	case *eth.SignedBeaconBlockAltair:
		return WrappedAltairSignedBeaconBlock(b)
	case *eth.SignedBeaconBlockBellatrix:
		return WrappedBellatrixSignedBeaconBlock(b)
	default:
		return nil, errors.Wrapf(ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// WrappedBeaconBlock will wrap a signed beacon block to conform to the
// signed beacon block interface.
func WrappedBeaconBlock(i interface{}) (block.BeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericBeaconBlock_Phase0:
		return WrappedPhase0BeaconBlock(b.Phase0), nil
	case *eth.BeaconBlock:
		return WrappedPhase0BeaconBlock(b), nil
	case *eth.GenericBeaconBlock_Altair:
		return WrappedAltairBeaconBlock(b.Altair)
	case *eth.BeaconBlockAltair:
		return WrappedAltairBeaconBlock(b)
	case *eth.GenericBeaconBlock_Bellatrix:
		return WrappedBellatrixBeaconBlock(b.Bellatrix)
	case *eth.BeaconBlockBellatrix:
		return WrappedBellatrixBeaconBlock(b)
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// BuildSignedBeaconBlock assembles a block.SignedBeaconBlock interface compatible struct from a
// given beacon block an the appropriate signature. This method may be used to easily create a
// signed beacon block.
func BuildSignedBeaconBlock(blk block.BeaconBlock, signature []byte) (block.SignedBeaconBlock, error) {
	switch b := blk.(type) {
	case Phase0BeaconBlock:
		pb, ok := b.Proto().(*eth.BeaconBlock)
		if !ok {
			return nil, errors.New("unable to access inner phase0 proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlock{Block: pb, Signature: signature})
	case altairBeaconBlock:
		pb, ok := b.Proto().(*eth.BeaconBlockAltair)
		if !ok {
			return nil, errors.New("unable to access inner altair proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockAltair{Block: pb, Signature: signature})
	case bellatrixBeaconBlock:
		pb, ok := b.Proto().(*eth.BeaconBlockBellatrix)
		if !ok {
			return nil, errors.New("unable to access inner bellatrix proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockBellatrix{Block: pb, Signature: signature})
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", b)
	}
}

func UnwrapGenericSignedBeaconBlock(gb *eth.GenericSignedBeaconBlock) (block.SignedBeaconBlock, error) {
	if gb == nil {
		return nil, ErrNilObjectWrapped
	}
	switch bb := gb.Block.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return WrappedSignedBeaconBlock(bb.Phase0)
	case *eth.GenericSignedBeaconBlock_Altair:
		return WrappedSignedBeaconBlock(bb.Altair)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return WrappedSignedBeaconBlock(bb.Bellatrix)
	default:
		return nil, errors.Wrapf(ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", gb)
	}
}
