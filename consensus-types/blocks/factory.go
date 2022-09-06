package blocks

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

var (
	// ErrUnsupportedSignedBeaconBlock is returned when the struct type is not a supported signed
	// beacon block type.
	ErrUnsupportedSignedBeaconBlock = errors.New("unsupported signed beacon block")
	// errUnsupportedBeaconBlock is returned when the struct type is not a supported beacon block
	// type.
	errUnsupportedBeaconBlock = errors.New("unsupported beacon block")
	// errUnsupportedBeaconBlockBody is returned when the struct type is not a supported beacon block body
	// type.
	errUnsupportedBeaconBlockBody = errors.New("unsupported beacon block body")
	// ErrNilObject is returned in a constructor when the underlying object is nil.
	ErrNilObject = errors.New("received nil object")
	// ErrNilSignedBeaconBlock is returned when a nil signed beacon block is received.
	ErrNilSignedBeaconBlock = errors.New("signed beacon block can't be nil")
)

// NewSignedBeaconBlock creates a signed beacon block from a protobuf signed beacon block.
func NewSignedBeaconBlock(i interface{}) (interfaces.SignedBeaconBlock, error) {
	switch b := i.(type) {
	case nil:
		return nil, ErrNilObject
	case *eth.GenericSignedBeaconBlock_Phase0:
		return initSignedBlockFromProtoPhase0(b.Phase0)
	case *eth.SignedBeaconBlock:
		return initSignedBlockFromProtoPhase0(b)
	case *eth.GenericSignedBeaconBlock_Altair:
		return initSignedBlockFromProtoAltair(b.Altair)
	case *eth.SignedBeaconBlockAltair:
		return initSignedBlockFromProtoAltair(b)
	case *eth.GenericSignedBeaconBlock_Bellatrix:
		return initSignedBlockFromProtoBellatrix(b.Bellatrix)
	case *eth.SignedBeaconBlockBellatrix:
		return initSignedBlockFromProtoBellatrix(b)
	case *eth.GenericSignedBeaconBlock_BlindedBellatrix:
		return initBlindedSignedBlockFromProtoBellatrix(b.BlindedBellatrix)
	case *eth.SignedBlindedBeaconBlockBellatrix:
		return initBlindedSignedBlockFromProtoBellatrix(b)
	default:
		return nil, errors.Wrapf(ErrUnsupportedSignedBeaconBlock, "unable to create block from type %T", i)
	}
}

// NewBeaconBlock creates a beacon block from a protobuf beacon block.
func NewBeaconBlock(i interface{}) (interfaces.BeaconBlock, error) {
	switch b := i.(type) {
	case nil:
		return nil, ErrNilObject
	case *eth.GenericBeaconBlock_Phase0:
		return initBlockFromProtoPhase0(b.Phase0)
	case *eth.BeaconBlock:
		return initBlockFromProtoPhase0(b)
	case *eth.GenericBeaconBlock_Altair:
		return initBlockFromProtoAltair(b.Altair)
	case *eth.BeaconBlockAltair:
		return initBlockFromProtoAltair(b)
	case *eth.GenericBeaconBlock_Bellatrix:
		return initBlockFromProtoBellatrix(b.Bellatrix)
	case *eth.BeaconBlockBellatrix:
		return initBlockFromProtoBellatrix(b)
	case *eth.GenericBeaconBlock_BlindedBellatrix:
		return initBlindedBlockFromProtoBellatrix(b.BlindedBellatrix)
	case *eth.BlindedBeaconBlockBellatrix:
		return initBlindedBlockFromProtoBellatrix(b)
	default:
		return nil, errors.Wrapf(errUnsupportedBeaconBlock, "unable to create block from type %T", i)
	}
}

// NewBeaconBlockBody creates a beacon block body from a protobuf beacon block body.
func NewBeaconBlockBody(i interface{}) (interfaces.BeaconBlockBody, error) {
	switch b := i.(type) {
	case nil:
		return nil, ErrNilObject
	case *eth.BeaconBlockBody:
		return initBlockBodyFromProtoPhase0(b)
	case *eth.BeaconBlockBodyAltair:
		return initBlockBodyFromProtoAltair(b)
	case *eth.BeaconBlockBodyBellatrix:
		return initBlockBodyFromProtoBellatrix(b)
	case *eth.BlindedBeaconBlockBodyBellatrix:
		return initBlindedBlockBodyFromProtoBellatrix(b)
	default:
		return nil, errors.Wrapf(errUnsupportedBeaconBlockBody, "unable to create block body from type %T", i)
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
			return nil, errIncorrectBlockVersion
		}
		return NewSignedBeaconBlock(&eth.SignedBeaconBlock{Block: pb, Signature: signature})
	case version.Altair:
		pb, ok := pb.(*eth.BeaconBlockAltair)
		if !ok {
			return nil, errIncorrectBlockVersion
		}
		return NewSignedBeaconBlock(&eth.SignedBeaconBlockAltair{Block: pb, Signature: signature})
	case version.Bellatrix:
		if blk.IsBlinded() {
			pb, ok := pb.(*eth.BlindedBeaconBlockBellatrix)
			if !ok {
				return nil, errIncorrectBlockVersion
			}
			return NewSignedBeaconBlock(&eth.SignedBlindedBeaconBlockBellatrix{Block: pb, Signature: signature})
		}
		pb, ok := pb.(*eth.BeaconBlockBellatrix)
		if !ok {
			return nil, errIncorrectBlockVersion
		}
		return NewSignedBeaconBlock(&eth.SignedBeaconBlockBellatrix{Block: pb, Signature: signature})
	default:
		return nil, errUnsupportedBeaconBlock
	}
}

// BuildSignedBeaconBlockFromExecutionPayload takes a signed, blinded beacon block and converts into
// a full, signed beacon block by specifying an execution payload.
func BuildSignedBeaconBlockFromExecutionPayload(
	blk interfaces.SignedBeaconBlock, payload *enginev1.ExecutionPayload,
) (interfaces.SignedBeaconBlock, error) {
	if err := BeaconBlockIsNil(blk); err != nil {
		return nil, err
	}
	b := blk.Block()
	payloadHeader, err := b.Body().Execution()
	switch {
	case errors.Is(err, ErrUnsupportedGetter):
		return nil, errors.Wrap(err, "can only build signed beacon block from blinded format")
	case err != nil:
		return nil, errors.Wrap(err, "could not get execution payload header")
	default:
	}
	wrappedPayload, err := WrappedExecutionPayload(payload)
	if err != nil {
		return nil, err
	}
	empty, err := IsEmptyExecutionData(wrappedPayload)
	if err != nil {
		return nil, err
	}
	if !empty {
		payloadRoot, err := payload.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash tree root execution payload")
		}
		payloadHeaderRoot, err := payloadHeader.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash tree root payload header")
		}
		if payloadRoot != payloadHeaderRoot {
			return nil, fmt.Errorf(
				"payload %#x and header %#x roots do not match",
				payloadRoot,
				payloadHeaderRoot,
			)
		}
	}
	syncAgg, err := b.Body().SyncAggregate()
	if err != nil {
		return nil, errors.Wrap(err, "could not get sync aggregate from block body")
	}
	parentRoot := b.ParentRoot()
	stateRoot := b.StateRoot()
	randaoReveal := b.Body().RandaoReveal()
	graffiti := b.Body().Graffiti()
	sig := blk.Signature()
	bellatrixFullBlock := &eth.SignedBeaconBlockBellatrix{
		Block: &eth.BeaconBlockBellatrix{
			Slot:          b.Slot(),
			ProposerIndex: b.ProposerIndex(),
			ParentRoot:    parentRoot[:],
			StateRoot:     stateRoot[:],
			Body: &eth.BeaconBlockBodyBellatrix{
				RandaoReveal:      randaoReveal[:],
				Eth1Data:          b.Body().Eth1Data(),
				Graffiti:          graffiti[:],
				ProposerSlashings: b.Body().ProposerSlashings(),
				AttesterSlashings: b.Body().AttesterSlashings(),
				Attestations:      b.Body().Attestations(),
				Deposits:          b.Body().Deposits(),
				VoluntaryExits:    b.Body().VoluntaryExits(),
				SyncAggregate:     syncAgg,
				ExecutionPayload:  payload,
			},
		},
		Signature: sig[:],
	}
	return NewSignedBeaconBlock(bellatrixFullBlock)
}
