package consensus_types

import (
	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"google.golang.org/protobuf/proto"
)

// SSZItem defines a struct which provides Marshal,
// Unmarshal, and HashTreeRoot SSZ operations.
type SSZItem interface {
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
}

// Container defines the base methods required for a consensus
// data structure used in Prysm, containing utilities for SSZ
// as well as conversion methods to a protobuf representation for use
// with Prysm's gRPC API.
type Container interface {
	SSZItem
	IsNil() bool
	Proto() proto.Message
	FromProto(m proto.Message)
}

// SignedBeaconBlock describes the method set of a signed beacon block.
type SignedBeaconBlock interface {
	Container
	Block() BeaconBlock
	Signature() []byte
	Copy() SignedBeaconBlock
	PbGenericBlock() (*ethpb.GenericSignedBeaconBlock, error)
	PbPhase0Block() (*ethpb.SignedBeaconBlock, error)
	PbAltairBlock() (*ethpb.SignedBeaconBlockAltair, error)
	PbBellatrixBlock() (*ethpb.SignedBeaconBlockBellatrix, error)
	PbBlindedBellatrixBlock() (*ethpb.SignedBlindedBeaconBlockBellatrix, error)
	Header() (*ethpb.SignedBeaconBlockHeader, error)
}

// BeaconBlock describes an interface which states the methods
// employed by an object that is a beacon block.
type BeaconBlock interface {
	Container
	Slot() types.Slot
	ProposerIndex() types.ValidatorIndex
	ParentRoot() []byte
	StateRoot() []byte
	Body() BeaconBlockBody
	AsSignRequestObject() validatorpb.SignRequestObject
}

// BeaconBlockBody describes the method set employed by an object
// that is a beacon block body.
type BeaconBlockBody interface {
	Container
	RandaoReveal() []byte
	Eth1Data() *ethpb.Eth1Data
	Graffiti() []byte
	ProposerSlashings() []*ethpb.ProposerSlashing
	AttesterSlashings() []*ethpb.AttesterSlashing
	Attestations() []*ethpb.Attestation
	Deposits() []*ethpb.Deposit
	VoluntaryExits() []*ethpb.SignedVoluntaryExit
	SyncAggregate() (*ethpb.SyncAggregate, error)
	ExecutionPayload() (*enginev1.ExecutionPayload, error)
	ExecutionPayloadHeader() (*ethpb.ExecutionPayloadHeader, error)
}
