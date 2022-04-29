package interfaces

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"google.golang.org/protobuf/proto"
)

// SignedBeaconBlock is an interface describing the method set of
// a signed beacon block.
type SignedBeaconBlock interface {
	Block() BeaconBlock
	Signature() []byte
	IsNil() bool
	Copy() SignedBeaconBlock
	Proto() proto.Message
	PbGenericBlock() (*ethpb.GenericSignedBeaconBlock, error)
	PbPhase0Block() (*ethpb.SignedBeaconBlock, error)
	PbAltairBlock() (*ethpb.SignedBeaconBlockAltair, error)
	PbBellatrixBlock() (*ethpb.SignedBeaconBlockBellatrix, error)
	PbBlindedBellatrixBlock() (*ethpb.SignedBlindedBeaconBlockBellatrix, error)
	ssz.Marshaler
	ssz.Unmarshaler
	Version() int
	Header() (*ethpb.SignedBeaconBlockHeader, error)
}

// BeaconBlock describes an interface which states the methods
// employed by an object that is a beacon block.
type BeaconBlock interface {
	Slot() types.Slot
	ProposerIndex() types.ValidatorIndex
	ParentRoot() []byte
	StateRoot() []byte
	Body() BeaconBlockBody
	IsNil() bool
	IsBlinded() bool
	HashTreeRoot() ([32]byte, error)
	Proto() proto.Message
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	AsSignRequestObject() validatorpb.SignRequestObject
}

// BeaconBlockBody describes the method set employed by an object
// that is a beacon block body.
type BeaconBlockBody interface {
	RandaoReveal() []byte
	Eth1Data() *ethpb.Eth1Data
	Graffiti() []byte
	ProposerSlashings() []*ethpb.ProposerSlashing
	AttesterSlashings() []*ethpb.AttesterSlashing
	Attestations() []*ethpb.Attestation
	Deposits() []*ethpb.Deposit
	VoluntaryExits() []*ethpb.SignedVoluntaryExit
	SyncAggregate() (*ethpb.SyncAggregate, error)
	IsNil() bool
	HashTreeRoot() ([32]byte, error)
	Proto() proto.Message
	ExecutionPayload() (*enginev1.ExecutionPayload, error)
	ExecutionPayloadHeader() (*ethpb.ExecutionPayloadHeader, error)
}

// SignedBeaconBlockHeaderFromBlock function to retrieve signed block header from block.
func SignedBeaconBlockHeaderFromBlock(block *ethpb.SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	if block.Block == nil || block.Block.Body == nil {
		return nil, errors.New("nil block")
	}

	bodyRoot, err := block.Block.Body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          block.Block.Slot,
			ProposerIndex: block.Block.ProposerIndex,
			ParentRoot:    block.Block.ParentRoot,
			StateRoot:     block.Block.StateRoot,
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature,
	}, nil
}

// SignedBeaconBlockHeaderFromBlockInterface function to retrieve signed block header from block.
func SignedBeaconBlockHeaderFromBlockInterface(sb SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	b := sb.Block()
	if b.IsNil() || b.Body().IsNil() {
		return nil, errors.New("nil block")
	}

	h, err := BeaconBlockHeaderFromBlockInterface(b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block header of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Header:    h,
		Signature: sb.Signature(),
	}, nil
}

// BeaconBlockHeaderFromBlock function to retrieve block header from block.
func BeaconBlockHeaderFromBlock(block *ethpb.BeaconBlock) (*ethpb.BeaconBlockHeader, error) {
	if block.Body == nil {
		return nil, errors.New("nil block body")
	}

	bodyRoot, err := block.Body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.BeaconBlockHeader{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
		BodyRoot:      bodyRoot[:],
	}, nil
}

// BeaconBlockHeaderFromBlockInterface function to retrieve block header from block.
func BeaconBlockHeaderFromBlockInterface(block BeaconBlock) (*ethpb.BeaconBlockHeader, error) {
	if block.Body().IsNil() {
		return nil, errors.New("nil block body")
	}

	bodyRoot, err := block.Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.BeaconBlockHeader{
		Slot:          block.Slot(),
		ProposerIndex: block.ProposerIndex(),
		ParentRoot:    block.ParentRoot(),
		StateRoot:     block.StateRoot(),
		BodyRoot:      bodyRoot[:],
	}, nil
}
