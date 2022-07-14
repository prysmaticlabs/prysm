package wrapper

import (
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

var (
	_ = interfaces.SignedBeaconBlock(&bellatrixSignedBeaconBlock{})
	_ = interfaces.BeaconBlock(&bellatrixBeaconBlock{})
	_ = interfaces.BeaconBlockBody(&bellatrixBeaconBlockBody{})
)

// bellatrixSignedBeaconBlock is a convenience wrapper around a Bellatrix blinded beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type bellatrixSignedBeaconBlock struct {
	b *eth.SignedBeaconBlockBellatrix
}

// wrappedBellatrixSignedBeaconBlock is a constructor which wraps a protobuf Bellatrix block with the block wrapper.
func wrappedBellatrixSignedBeaconBlock(b *eth.SignedBeaconBlockBellatrix) (interfaces.SignedBeaconBlock, error) {
	w := bellatrixSignedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w bellatrixSignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w bellatrixSignedBeaconBlock) Block() interfaces.BeaconBlock {
	return bellatrixBeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w bellatrixSignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w bellatrixSignedBeaconBlock) Copy() interfaces.SignedBeaconBlock {
	return bellatrixSignedBeaconBlock{b: eth.CopySignedBeaconBlockBellatrix(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w bellatrixSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (w bellatrixSignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (w bellatrixSignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
// form.
func (w bellatrixSignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w bellatrixSignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w bellatrixSignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: w.b},
	}, nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (w bellatrixSignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return w.b, nil
}

// PbBlindedBellatrixBlock is a stub.
func (bellatrixSignedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	return nil, ErrUnsupportedBlindedBellatrixBlock
}

// PbPhase0Block is a stub.
func (bellatrixSignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (bellatrixSignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, ErrUnsupportedAltairBlock
}

func (w bellatrixSignedBeaconBlock) ToBlinded() (interfaces.SignedBeaconBlock, error) {
	if w.Block().IsNil() {
		return nil, errors.New("cannot convert nil block to blinded format")
	}
	payload := w.b.Block.Body.ExecutionPayload
	wrappedPayload, err := WrappedExecutionPayload(payload)
	if err != nil {
		return nil, err
	}
	header, err := PayloadToHeader(wrappedPayload)
	if err != nil {
		return nil, err
	}
	return signedBlindedBeaconBlockBellatrix{
		b: &eth.SignedBlindedBeaconBlockBellatrix{
			Block: &eth.BlindedBeaconBlockBellatrix{
				Slot:          w.b.Block.Slot,
				ProposerIndex: w.b.Block.ProposerIndex,
				ParentRoot:    w.b.Block.ParentRoot,
				StateRoot:     w.b.Block.StateRoot,
				Body: &eth.BlindedBeaconBlockBodyBellatrix{
					RandaoReveal:           w.b.Block.Body.RandaoReveal,
					Eth1Data:               w.b.Block.Body.Eth1Data,
					Graffiti:               w.b.Block.Body.Graffiti,
					ProposerSlashings:      w.b.Block.Body.ProposerSlashings,
					AttesterSlashings:      w.b.Block.Body.AttesterSlashings,
					Attestations:           w.b.Block.Body.Attestations,
					Deposits:               w.b.Block.Body.Deposits,
					VoluntaryExits:         w.b.Block.Body.VoluntaryExits,
					SyncAggregate:          w.b.Block.Body.SyncAggregate,
					ExecutionPayloadHeader: header,
				},
			},
			Signature: w.b.Signature,
		},
	}, nil
}

// Version of the underlying protobuf object.
func (bellatrixSignedBeaconBlock) Version() int {
	return version.Bellatrix
}

func (w bellatrixSignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
	root, err := w.b.Block.Body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not hash block")
	}

	return &eth.SignedBeaconBlockHeader{
		Header: &eth.BeaconBlockHeader{
			Slot:          w.b.Block.Slot,
			ProposerIndex: w.b.Block.ProposerIndex,
			ParentRoot:    w.b.Block.ParentRoot,
			StateRoot:     w.b.Block.StateRoot,
			BodyRoot:      root[:],
		},
		Signature: w.Signature(),
	}, nil
}

// bellatrixBeaconBlock is the wrapper for the actual block.
type bellatrixBeaconBlock struct {
	b *eth.BeaconBlockBellatrix
}

// wrappedBellatrixBeaconBlock is a constructor which wraps a protobuf Bellatrix object
// with the block wrapper.
func wrappedBellatrixBeaconBlock(b *eth.BeaconBlockBellatrix) (interfaces.BeaconBlock, error) {
	w := bellatrixBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w bellatrixBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns the proposer index of the beacon block.
func (w bellatrixBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w bellatrixBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w bellatrixBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w bellatrixBeaconBlock) Body() interfaces.BeaconBlockBody {
	return bellatrixBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w bellatrixBeaconBlock) IsNil() bool {
	return w.b == nil
}

// IsBlinded checks if the beacon block is a blinded block.
func (bellatrixBeaconBlock) IsBlinded() bool {
	return false
}

// HashTreeRoot returns the ssz root of the block.
func (w bellatrixBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w bellatrixBeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w bellatrixBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block's ssz
// form to the provided byte buffer.
func (w bellatrixBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized block.
func (w bellatrixBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the beacon block from its relevant ssz
// form.
func (w bellatrixBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w bellatrixBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (bellatrixBeaconBlock) Version() int {
	return version.Bellatrix
}

// AsSignRequestObject returns the underlying sign request object.
func (w bellatrixBeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlockV3{
		BlockV3: w.b,
	}
}

// bellatrixBeaconBlockBody is a wrapper of a beacon block body.
type bellatrixBeaconBlockBody struct {
	b *eth.BeaconBlockBodyBellatrix
}

// wrappedBellatrixBeaconBlockBody is a constructor which wraps a protobuf bellatrix object
// with the block wrapper.
func wrappedBellatrixBeaconBlockBody(b *eth.BeaconBlockBodyBellatrix) (interfaces.BeaconBlockBody, error) {
	w := bellatrixBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w bellatrixBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w bellatrixBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w bellatrixBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w bellatrixBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w bellatrixBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w bellatrixBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w bellatrixBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w bellatrixBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w bellatrixBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w bellatrixBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w bellatrixBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w bellatrixBeaconBlockBody) Proto() proto.Message {
	return w.b
}

// Execution returns the Execution payload of the block body.
func (w bellatrixBeaconBlockBody) Execution() (interfaces.ExecutionData, error) {
	return WrappedExecutionPayload(w.b.ExecutionPayload)
}
