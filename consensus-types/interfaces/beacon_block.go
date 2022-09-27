package interfaces

import (
	ssz "github.com/prysmaticlabs/fastssz"
	field_params "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// SignedBeaconBlock is an interface describing the method set of
// a signed beacon block.
type SignedBeaconBlock interface {
	Block() BeaconBlock
	Signature() [field_params.BLSSignatureLength]byte
	IsNil() bool
	Copy() (SignedBeaconBlock, error)
	Proto() (proto.Message, error)
	PbGenericBlock() (*ethpb.GenericSignedBeaconBlock, error)
	PbPhase0Block() (*ethpb.SignedBeaconBlock, error)
	PbAltairBlock() (*ethpb.SignedBeaconBlockAltair, error)
	ToBlinded() (SignedBeaconBlock, error)
	PbBellatrixBlock() (*ethpb.SignedBeaconBlockBellatrix, error)
	PbBlindedBellatrixBlock() (*ethpb.SignedBlindedBeaconBlockBellatrix, error)
	PbEip4844Block() (*ethpb.SignedBeaconBlockWithBlobKZGs, error)
	ssz.Marshaler
	ssz.Unmarshaler
	Version() int
	IsBlinded() bool
	Header() (*ethpb.SignedBeaconBlockHeader, error)
}

// BeaconBlock describes an interface which states the methods
// employed by an object that is a beacon block.
type BeaconBlock interface {
	Slot() types.Slot
	ProposerIndex() types.ValidatorIndex
	ParentRoot() [field_params.RootLength]byte
	StateRoot() [field_params.RootLength]byte
	Body() BeaconBlockBody
	IsNil() bool
	IsBlinded() bool
	HashTreeRoot() ([field_params.RootLength]byte, error)
	Proto() (proto.Message, error)
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	Version() int
	AsSignRequestObject() (validatorpb.SignRequestObject, error)
}

// BeaconBlockBody describes the method set employed by an object
// that is a beacon block body.
type BeaconBlockBody interface {
	RandaoReveal() [field_params.BLSSignatureLength]byte
	Eth1Data() *ethpb.Eth1Data
	Graffiti() [field_params.RootLength]byte
	ProposerSlashings() []*ethpb.ProposerSlashing
	AttesterSlashings() []*ethpb.AttesterSlashing
	Attestations() []*ethpb.Attestation
	Deposits() []*ethpb.Deposit
	VoluntaryExits() []*ethpb.SignedVoluntaryExit
	SyncAggregate() (*ethpb.SyncAggregate, error)
	IsNil() bool
	HashTreeRoot() ([32]byte, error)
	Proto() (proto.Message, error)
	BlobKzgs() ([][]byte, error)
	Execution() (WrappedExecutionData, error)
}

// ExecutionData represents execution layer information that is contained
// within post-Bellatrix beacon block bodies. ONLY the fields that are on EVERY
// type of ExecutionPayload are here. Header and EIP-4844 fields can't be here.
type ExecutionData interface {
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	ProtoReflect() protoreflect.Message
	GetParentHash() []byte
	GetFeeRecipient() []byte
	GetStateRoot() []byte
	GetReceiptsRoot() []byte
	GetLogsBloom() []byte
	GetPrevRandao() []byte
	GetBlockNumber() uint64
	GetGasLimit() uint64
	GetGasUsed() uint64
	GetTimestamp() uint64
	GetExtraData() []byte
	GetBaseFeePerGas() []byte
	GetBlockHash() []byte
}

type Wrapped interface {
	IsNil() bool
	Proto() proto.Message
	ExcessBlobs
}

type ExcessBlobs interface {
	GetExcessBlobs() (uint64, error) // Only on EIP-4844 blocks -- both payload and header
}

type ExecutionPayloadHeader interface {
	ExecutionData
	GetTransactionsRoot() []byte
}

type WrappedExecutionPayloadHeader interface {
	Wrapped
	ExecutionPayloadHeader
}

// Can be holding either the full ExecutionPayload or an ExecutionPayloadHeader
// and either the legacy format or the 4844 format including ExcessBlobs.
type WrappedExecutionData interface {
	Wrapped
	ExecutionData
	ToHeader() (WrappedExecutionPayloadHeader, error)

	// Optional, can error!
	GetTransactions() ([][]byte, error) // Only on payload, absent on header
	GetTransactionsRoot() ([]byte, error) // Present as a field on header, computed on payload
}
