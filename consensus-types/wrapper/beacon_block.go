package wrapper

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var (
	// ErrUnsupportedField is returned when a field is not supported by a specific beacon block type.
	// This allows us to create a generic beacon block interface that is implemented by different
	// fork versions of beacon blocks.
	ErrUnsupportedField = errors.New("unsupported field for block type")
	// ErrUnsupportedSignedBeaconBlock is returned when the struct type is not a supported signed
	// beacon block type.
	ErrUnsupportedSignedBeaconBlock = errors.New("unsupported signed beacon block")
	// ErrUnsupportedBeaconBlock is returned when the struct type is not a supported beacon block
	// type.
	ErrUnsupportedBeaconBlock = errors.New("unsupported beacon block")
	// ErrUnsupportedBeaconBlockBody is returned when the struct type is not a supported beacon block body
	// type.
	ErrUnsupportedBeaconBlockBody = errors.New("unsupported beacon block body")
	// ErrUnsupportedPhase0Block is returned when accessing a phase0 block from a non-phase0 wrapped
	// block.
	ErrUnsupportedPhase0Block = errors.New("unsupported phase0 block")
	// ErrUnsupportedAltairBlock is returned when accessing an altair block from non-altair wrapped
	// block.
	ErrUnsupportedAltairBlock = errors.New("unsupported altair block")
	// ErrUnsupportedBellatrixBlock is returned when accessing a bellatrix block from a non-bellatrix wrapped
	// block.
	ErrUnsupportedBellatrixBlock = errors.New("unsupported bellatrix block")
	// ErrUnsupportedBlindedBellatrixBlock is returned when accessing a blinded bellatrix block from unsupported method.
	ErrUnsupportedBlindedBellatrixBlock = errors.New("unsupported blinded bellatrix block")
	// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
	ErrNilObjectWrapped     = errors.New("attempted to wrap nil object")
	ErrNilSignedBeaconBlock = errors.New("signed beacon block can't be nil")
	ErrNilBeaconBlock       = errors.New("beacon block can't be nil")
	ErrNilBeaconBlockBody   = errors.New("beacon block body can't be nil")
)

// WrappedSignedBeaconBlock will wrap a signed beacon block to conform to the
// signed beacon block interface.
func WrappedSignedBeaconBlock(i interface{}) (interfaces.SignedBeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return wrappedPhase0SignedBeaconBlock(b.Phase0), nil
	case *eth.SignedBeaconBlock:
		return wrappedPhase0SignedBeaconBlock(b), nil
	case *eth.GenericSignedBeaconBlock_Altair:
		return wrappedAltairSignedBeaconBlock(b.Altair)
	case *eth.SignedBeaconBlockAltair:
		return wrappedAltairSignedBeaconBlock(b)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return wrappedBellatrixSignedBeaconBlock(b.Bellatrix)
	case *eth.SignedBeaconBlockBellatrix:
		return wrappedBellatrixSignedBeaconBlock(b)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return wrappedBellatrixSignedBlindedBeaconBlock(b.BlindedBellatrix)
	case *eth.SignedBlindedBeaconBlockBellatrix:
		return wrappedBellatrixSignedBlindedBeaconBlock(b)
	case nil:
		return nil, ErrNilObjectWrapped
	default:
		return nil, errors.Wrapf(ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// WrappedBeaconBlock will wrap a beacon block to conform to the
// beacon block interface.
func WrappedBeaconBlock(i interface{}) (interfaces.BeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericBeaconBlock_Phase0:
		return wrappedPhase0BeaconBlock(b.Phase0), nil
	case *eth.BeaconBlock:
		return wrappedPhase0BeaconBlock(b), nil
	case *eth.GenericBeaconBlock_Altair:
		return wrappedAltairBeaconBlock(b.Altair)
	case *eth.BeaconBlockAltair:
		return wrappedAltairBeaconBlock(b)
	case *eth.GenericBeaconBlock_Bellatrix:
		return wrappedBellatrixBeaconBlock(b.Bellatrix)
	case *eth.BeaconBlockBellatrix:
		return wrappedBellatrixBeaconBlock(b)
	case *eth.GenericBeaconBlock_BlindedBellatrix:
		return wrappedBellatrixBlindedBeaconBlock(b.BlindedBellatrix)
	case *eth.BlindedBeaconBlockBellatrix:
		return wrappedBellatrixBlindedBeaconBlock(b)
	case nil:
		return nil, ErrNilObjectWrapped
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// WrappedBeaconBlockBody will wrap a beacon block body to conform to the
// beacon block interface.
func WrappedBeaconBlockBody(i interface{}) (interfaces.BeaconBlockBody, error) {
	switch b := i.(type) {
	case *eth.BeaconBlockBody:
		return wrappedPhase0BeaconBlockBody(b), nil
	case *eth.BeaconBlockBodyAltair:
		return wrappedAltairBeaconBlockBody(b)
	case *eth.BeaconBlockBodyBellatrix:
		return wrappedBellatrixBeaconBlockBody(b)
	case *eth.BlindedBeaconBlockBodyBellatrix:
		return wrappedBellatrixBlindedBeaconBlockBody(b)
	case nil:
		return nil, ErrNilObjectWrapped
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlockBody, "unable to wrap block body of type %T", i)
	}
}

// BuildSignedBeaconBlock assembles a block.SignedBeaconBlock interface compatible struct from a
// given beacon block an the appropriate signature. This method may be used to easily create a
// signed beacon block.
func BuildSignedBeaconBlock(blk interfaces.BeaconBlock, signature []byte) (interfaces.SignedBeaconBlock, error) {
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
	case blindedBeaconBlockBellatrix:
		pb, ok := b.Proto().(*eth.BlindedBeaconBlockBellatrix)
		if !ok {
			return nil, errors.New("unable to access inner bellatrix proto")
		}
		return WrappedSignedBeaconBlock(&eth.SignedBlindedBeaconBlockBellatrix{Block: pb, Signature: signature})
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", b)
	}
}

func UnwrapGenericSignedBeaconBlock(gb *eth.GenericSignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
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
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return WrappedSignedBeaconBlock(bb.BlindedBellatrix)
	default:
		return nil, errors.Wrapf(ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", gb)
	}
}

// BeaconBlockIsNil checks if any composite field of input signed beacon block is nil.
// Access to these nil fields will result in run time panic,
// it is recommended to run these checks as first line of defense.
func BeaconBlockIsNil(b interfaces.SignedBeaconBlock) error {
	if b == nil || b.IsNil() {
		return ErrNilSignedBeaconBlock
	}
	if b.Block().IsNil() {
		return ErrNilBeaconBlock
	}
	if b.Block().Body().IsNil() {
		return ErrNilBeaconBlockBody
	}
	return nil
}
