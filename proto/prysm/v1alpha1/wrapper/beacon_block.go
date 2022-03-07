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

	// ErrUnsupportedPhase0Block is returned when accessing a phase0 block from an altair wrapped
	// block.
	ErrUnsupportedPhase0Block = errors.New("unsupported phase0 block")
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
	case *eth.SignedBeaconBlockWithBlobKZGs:
		return WrappedEip4844SignedBeaconBlock(b)
	default:
		return nil, errors.Wrapf(ErrUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// WrappedBeaconBlock will wrap a signed beacon block to conform to the
// signed beacon block interface.
func WrappedBeaconBlock(i interface{}) (block.BeaconBlock, error) {
	switch b := i.(type) {
	case *eth.BeaconBlock:
		return WrappedPhase0BeaconBlock(b), nil
	case *eth.BeaconBlockAltair:
		return WrappedAltairBeaconBlock(b)
	case *eth.BeaconBlockBellatrix:
		return WrappedBellatrixBeaconBlock(b)
	case *eth.BeaconBlockWithBlobKZGs:
		return WrappedEip4844BeaconBlock(b)
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

func BuildSignedBeaconBlock(blk interface{}, signature []byte) (block.SignedBeaconBlock, error) {
	switch b := blk.(type) {
	case *eth.BeaconBlock:
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlock{Block: b, Signature: signature})
	case *eth.BeaconBlockAltair:
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockAltair{Block: b, Signature: signature})
	case *eth.BeaconBlockBellatrix:
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockBellatrix{Block: b, Signature: signature})
	case *eth.BeaconBlockWithBlobKZGs:
		return WrappedSignedBeaconBlock(&eth.SignedBeaconBlockWithBlobKZGs{Block: b, Signature: signature})
	default:
		return nil, errors.Wrapf(ErrUnsupportedBeaconBlock, "unable to wrap block of type %T", b)
	}
}
