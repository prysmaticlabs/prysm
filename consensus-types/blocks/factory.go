package blocks

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

var (
	// errUnsupportedSignedBeaconBlock is returned when the struct type is not a supported signed
	// beacon block type.
	errUnsupportedSignedBeaconBlock = errors.New("unsupported signed beacon block")
	// errUnsupportedBeaconBlock is returned when the struct type is not a supported beacon block
	// type.
	errUnsupportedBeaconBlock = errors.New("unsupported beacon block")
	// errUnsupportedBeaconBlockBody is returned when the struct type is not a supported beacon block body
	// type.
	errUnsupportedBeaconBlockBody = errors.New("unsupported beacon block body")
	// errNilObjectWrapped is returned in a constructor when the underlying object is nil.
	errNilObjectWrapped     = errors.New("attempted to wrap nil object")
	errNilSignedBeaconBlock = errors.New("signed beacon block can't be nil")
	errNilBeaconBlock       = errors.New("beacon block can't be nil")
	errNilBeaconBlockBody   = errors.New("beacon block body can't be nil")
)

// NewSignedBeaconBlock creates a signed beacon block from a protobuf signed beacon block.
func NewSignedBeaconBlock(i interface{}) (interfaces.SignedBeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return InitSignedBlockFromProtoPhase0(b.Phase0)
	case *eth.SignedBeaconBlock:
		return InitSignedBlockFromProtoPhase0(b)
	case *eth.GenericSignedBeaconBlock_Altair:
		return InitSignedBlockFromProtoAltair(b.Altair)
	case *eth.SignedBeaconBlockAltair:
		return InitSignedBlockFromProtoAltair(b)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return InitSignedBlockFromProtoBellatrix(b.Bellatrix)
	case *eth.SignedBeaconBlockBellatrix:
		return InitSignedBlockFromProtoBellatrix(b)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return InitBlindedSignedBlockFromProtoBellatrix(b.BlindedBellatrix)
	case *eth.SignedBlindedBeaconBlockBellatrix:
		return InitBlindedSignedBlockFromProtoBellatrix(b)
	case nil:
		return nil, errNilObjectWrapped
	default:
		return nil, errors.Wrapf(errUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// NewBeaconBlock creates a beacon block from a protobuf beacon block.
func NewBeaconBlock(i interface{}) (interfaces.BeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericBeaconBlock_Phase0:
		return InitBlockFromProtoPhase0(b.Phase0)
	case *eth.BeaconBlock:
		return InitBlockFromProtoPhase0(b)
	case *eth.GenericBeaconBlock_Altair:
		return InitBlockFromProtoAltair(b.Altair)
	case *eth.BeaconBlockAltair:
		return InitBlockFromProtoAltair(b)
	case *eth.GenericBeaconBlock_Bellatrix:
		return InitBlockFromProtoBellatrix(b.Bellatrix)
	case *eth.BeaconBlockBellatrix:
		return InitBlockFromProtoBellatrix(b)
	case *eth.GenericBeaconBlock_BlindedBellatrix:
		return InitBlindedBlockFromProtoBellatrix(b.BlindedBellatrix)
	case *eth.BlindedBeaconBlockBellatrix:
		return InitBlindedBlockFromProtoBellatrix(b)
	case nil:
		return nil, errNilObjectWrapped
	default:
		return nil, errors.Wrapf(errUnsupportedBeaconBlock, "unable to wrap block of type %T", i)
	}
}

// NewBeaconBlockBody creates a beacon block body from a protobuf beacon block body.
func NewBeaconBlockBody(i interface{}) (interfaces.BeaconBlockBody, error) {
	switch b := i.(type) {
	case *eth.BeaconBlockBody:
		return InitBlockBodyFromProtoPhase0(b)
	case *eth.BeaconBlockBodyAltair:
		return InitBlockBodyFromProtoAltair(b)
	case *eth.BeaconBlockBodyBellatrix:
		return InitBlockBodyFromProtoBellatrix(b)
	case *eth.BlindedBeaconBlockBodyBellatrix:
		return InitBlindedBlockBodyFromProtoBellatrix(b)
	case nil:
		return nil, errNilObjectWrapped
	default:
		return nil, errors.Wrapf(errUnsupportedBeaconBlockBody, "unable to wrap block body of type %T", i)
	}
}

// BuildSignedBeaconBlock assembles a block.SignedBeaconBlock interface compatible struct from a
// given beacon block and the appropriate signature. This method may be used to easily create a
// signed beacon block.
func BuildSignedBeaconBlock(blk interfaces.BeaconBlock, signature []byte) (interfaces.SignedBeaconBlock, error) {
	pb, err := blk.Proto()
	if err != nil {
		return nil, err
	}
	switch blk.Version() {
	case version.Phase0:
		pb, ok := pb.(*eth.BeaconBlock)
		if !ok {
			return nil, errors.New("unable to access inner phase0 proto")
		}
		return NewSignedBeaconBlock(&eth.SignedBeaconBlock{Block: pb, Signature: signature})
	case version.Altair:
		pb, ok := pb.(*eth.BeaconBlockAltair)
		if !ok {
			return nil, errors.New("unable to access inner altair proto")
		}
		return NewSignedBeaconBlock(&eth.SignedBeaconBlockAltair{Block: pb, Signature: signature})
	case version.Bellatrix:
		pb, ok := pb.(*eth.BeaconBlockBellatrix)
		if !ok {
			return nil, errors.New("unable to access inner bellatrix proto")
		}
		return NewSignedBeaconBlock(&eth.SignedBeaconBlockBellatrix{Block: pb, Signature: signature})
	case version.BellatrixBlind:
		pb, ok := pb.(*eth.BlindedBeaconBlockBellatrix)
		if !ok {
			return nil, errors.New("unable to access inner bellatrix proto")
		}
		return NewSignedBeaconBlock(&eth.SignedBlindedBeaconBlockBellatrix{Block: pb, Signature: signature})
	default:
		return nil, errUnsupportedBeaconBlockBody
	}
}

// NewSignedBeaconBlockFromGeneric creates a signed beacon block
// from a protobuf generic signed beacon block.
func NewSignedBeaconBlockFromGeneric(gb *eth.GenericSignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
	if gb == nil {
		return nil, errNilObjectWrapped
	}
	switch bb := gb.Block.(type) {
	case *eth.GenericSignedBeaconBlock_Phase0:
		return NewSignedBeaconBlock(bb.Phase0)
	case *eth.GenericSignedBeaconBlock_Altair:
		return NewSignedBeaconBlock(bb.Altair)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return NewSignedBeaconBlock(bb.Bellatrix)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return NewSignedBeaconBlock(bb.BlindedBellatrix)
	default:
		return nil, errors.Wrapf(errUnsupportedSignedBeaconBlock, "unable to wrap block of type %T", gb)
	}
}

// BeaconBlockIsNil checks if any composite field of input signed beacon block is nil.
// Access to these nil fields will result in run time panic,
// it is recommended to run these checks as first line of defense.
func BeaconBlockIsNil(b interfaces.SignedBeaconBlock) error {
	if b == nil || b.IsNil() {
		return errNilSignedBeaconBlock
	}
	if b.Block().IsNil() {
		return errNilBeaconBlock
	}
	if b.Block().Body().IsNil() {
		return errNilBeaconBlockBody
	}
	return nil
}
