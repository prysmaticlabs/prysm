package bellatrix

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	typeerrors "github.com/prysmaticlabs/prysm/consensus-types/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	block "github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

// SignedBeaconBlock is a convenience wrapper around a Bellatrix blinded beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type SignedBeaconBlock struct {
	b *eth.SignedBeaconBlockBellatrix
}

// WrappedSignedBeaconBlock is a constructor which wraps a protobuf Bellatrix block with the block wrapper.
func WrappedSignedBeaconBlock(b *eth.SignedBeaconBlockBellatrix) (*SignedBeaconBlock, error) {
	w := &SignedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w SignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w SignedBeaconBlock) Block() interfaces.BeaconBlock {
	return &BeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w SignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w SignedBeaconBlock) Copy() interfaces.SignedBeaconBlock {
	return &SignedBeaconBlock{b: eth.CopySignedBeaconBlockBellatrix(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (w SignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (w SignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
// form.
func (w SignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w SignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w SignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: w.b},
	}, nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (w SignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return w.b, nil
}

// PbBlindedBellatrixBlock is a stub.
func (SignedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	return nil, typeerrors.ErrUnsupportedBlindedBellatrixBlock
}

// PbPhase0Block is a stub.
func (SignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, typeerrors.ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (SignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, typeerrors.ErrUnsupportedAltairBlock
}

// Version of the underlying protobuf object.
func (SignedBeaconBlock) Version() int {
	return version.Bellatrix
}

func (w SignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// BeaconBlock is the wrapper for the actual block.
type BeaconBlock struct {
	b *eth.BeaconBlockBellatrix
}

// WrappedBeaconBlock is a constructor which wraps a protobuf Bellatrix object
// with the block wrapper.
func WrappedBeaconBlock(b *eth.BeaconBlockBellatrix) (*BeaconBlock, error) {
	w := &BeaconBlock{b: b}
	if w.IsNil() {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w BeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns the proposer index of the beacon block.
func (w BeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w BeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w BeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w BeaconBlock) Body() interfaces.BeaconBlockBody {
	return &BeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w BeaconBlock) IsNil() bool {
	return w.b == nil
}

// IsBlinded checks if the beacon block is a blinded block.
func (BeaconBlock) IsBlinded() bool {
	return false
}

// HashTreeRoot returns the ssz root of the block.
func (w BeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w BeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w BeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block's ssz
// form to the provided byte buffer.
func (w BeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized block.
func (w BeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the beacon block from its relevant ssz
// form.
func (w BeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w BeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (BeaconBlock) Version() int {
	return version.Bellatrix
}

// AsSignRequestObject returns the underlying sign request object.
func (w BeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlockV3{
		BlockV3: w.b,
	}
}

// BeaconBlockBody is a wrapper of a beacon block body.
type BeaconBlockBody struct {
	b *eth.BeaconBlockBodyBellatrix
}

// WrappedBeaconBlockBody is a constructor which wraps a protobuf bellatrix object
// with the block wrapper.
func WrappedBeaconBlockBody(b *eth.BeaconBlockBodyBellatrix) (block.BeaconBlockBody, error) {
	w := BeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w BeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w BeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w BeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w BeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w BeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w BeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w BeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w BeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w BeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w BeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w BeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w BeaconBlockBody) Proto() proto.Message {
	return w.b
}

// ExecutionPayload returns the Execution payload of the block body.
func (w BeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return w.b.ExecutionPayload, nil
}

// ExecutionPayloadHeader is a stub.
func (w BeaconBlockBody) ExecutionPayloadHeader() (*eth.ExecutionPayloadHeader, error) {
	return nil, errors.Wrapf(typeerrors.ErrUnsupportedField, "ExecutionPayloadHeader for %T", w)
}
