package block

import (
	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
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
}
