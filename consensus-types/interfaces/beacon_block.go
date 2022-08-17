package interfaces

import (
	ssz "github.com/prysmaticlabs/fastssz"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"google.golang.org/protobuf/proto"
)

// SignedBeaconBlock is an interface describing the method set of
// a signed beacon block.
type SignedBeaconBlock interface {
	Block() BeaconBlock
	Signature() []byte
	IsNil() bool
	Copy() (SignedBeaconBlock, error)
	Proto() (proto.Message, error)
	PbGenericBlock() (*ethpb.GenericSignedBeaconBlock, error)
	PbPhase0Block() (*ethpb.SignedBeaconBlock, error)
	PbAltairBlock() (*ethpb.SignedBeaconBlockAltair, error)
	ToBlinded() (SignedBeaconBlock, error)
	PbBellatrixBlock() (*ethpb.SignedBeaconBlockBellatrix, error)
	PbBlindedBellatrixBlock() (*ethpb.SignedBlindedBeaconBlockBellatrix, error)
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
	ParentRoot() []byte
	StateRoot() []byte
	Body() BeaconBlockBody
	IsNil() bool
	IsBlinded() bool
	HashTreeRoot() ([32]byte, error)
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
	Proto() (proto.Message, error)
	Execution() (ExecutionData, error)
}

// ExecutionData represents execution layer information that is contained
// within post-Bellatrix beacon block bodies.
type ExecutionData interface {
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
	IsNil() bool
	Proto() proto.Message
	ParentHash() []byte
	FeeRecipient() []byte
	StateRoot() []byte
	ReceiptsRoot() []byte
	LogsBloom() []byte
	PrevRandao() []byte
	BlockNumber() uint64
	GasLimit() uint64
	GasUsed() uint64
	Timestamp() uint64
	ExtraData() []byte
	BaseFeePerGas() []byte
	BlockHash() []byte
	Transactions() ([][]byte, error)
}
