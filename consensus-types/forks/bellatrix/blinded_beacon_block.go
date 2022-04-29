package bellatrix

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	typeerrors "github.com/prysmaticlabs/prysm/consensus-types/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

// SignedBlindedBeaconBlock is a convenience wrapper around a Bellatrix blinded beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type SignedBlindedBeaconBlock struct {
	b *eth.SignedBlindedBeaconBlockBellatrix
}

// WrappedSignedBlindedBeaconBlock is a constructor which wraps a protobuf Bellatrix blinded block with the block wrapper.
func WrappedSignedBlindedBeaconBlock(b *eth.SignedBlindedBeaconBlockBellatrix) (*SignedBlindedBeaconBlock, error) {
	w := &SignedBlindedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w SignedBlindedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w SignedBlindedBeaconBlock) Block() interfaces.BeaconBlock {
	return &BlindedBeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w SignedBlindedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w SignedBlindedBeaconBlock) Copy() interfaces.SignedBeaconBlock {
	return &SignedBlindedBeaconBlock{b: eth.CopySignedBlindedBeaconBlockBellatrix(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w SignedBlindedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (w SignedBlindedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (w SignedBlindedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
// form.
func (w SignedBlindedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w SignedBlindedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w SignedBlindedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: w.b},
	}, nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (SignedBlindedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, typeerrors.ErrUnsupportedBellatrixBlock
}

// PbBlindedBellatrixBlock returns the underlying protobuf object.
func (w SignedBlindedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	return w.b, nil
}

// PbPhase0Block returns the underlying protobuf object.
func (SignedBlindedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, typeerrors.ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (SignedBlindedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, typeerrors.ErrUnsupportedAltairBlock
}

// Version of the underlying protobuf object.
func (SignedBlindedBeaconBlock) Version() int {
	return version.Bellatrix
}

// Header converts the underlying protobuf object from blinded block to header format.
func (w SignedBlindedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// BlindedBeaconBlock is the wrapper for the actual block.
type BlindedBeaconBlock struct {
	b *eth.BlindedBeaconBlockBellatrix
}

// WrappedBlindedBeaconBlock is a constructor which wraps a protobuf Bellatrix object
// with the block wrapper.
func WrappedBlindedBeaconBlock(b *eth.BlindedBeaconBlockBellatrix) (*BlindedBeaconBlock, error) {
	w := &BlindedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w BlindedBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns the proposer index of the beacon block.
func (w BlindedBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w BlindedBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w BlindedBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w BlindedBeaconBlock) Body() interfaces.BeaconBlockBody {
	return &BlindedBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w BlindedBeaconBlock) IsNil() bool {
	return w.b == nil
}

// IsBlinded checks if the beacon block is a blinded block.
func (BlindedBeaconBlock) IsBlinded() bool {
	return true
}

// HashTreeRoot returns the ssz root of the block.
func (w BlindedBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w BlindedBeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w BlindedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block's ssz
// form to the provided byte buffer.
func (w BlindedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized block.
func (w BlindedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the beacon block from its relevant ssz
// form.
func (w BlindedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w BlindedBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (BlindedBeaconBlock) Version() int {
	return version.Bellatrix
}

// AsSignRequestObject returns the underlying sign request object.
func (w BlindedBeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlindedBlockV3{
		BlindedBlockV3: w.b,
	}
}

// BlindedBeaconBlockBodyBellatrix is a wrapper of a beacon block body.
type BlindedBeaconBlockBody struct {
	b *eth.BlindedBeaconBlockBodyBellatrix
}

// WrappedBlindedBeaconBlockBody is a constructor which wraps a protobuf bellatrix object
// with the block wrapper.
func WrappedBlindedBeaconBlockBody(b *eth.BlindedBeaconBlockBodyBellatrix) (*BlindedBeaconBlockBody, error) {
	w := &BlindedBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, typeerrors.ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w BlindedBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w BlindedBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w BlindedBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w BlindedBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w BlindedBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w BlindedBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w BlindedBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w BlindedBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w BlindedBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w BlindedBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w BlindedBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w BlindedBeaconBlockBody) Proto() proto.Message {
	return w.b
}

// ExecutionPayload returns the execution payload of the block body.
func (w BlindedBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return nil, errors.Wrapf(typeerrors.ErrUnsupportedField, "ExecutionPayload for %T", w)
}

// ExecutionPayloadHeader returns the execution payload header of the block body.
func (w BlindedBeaconBlockBody) ExecutionPayloadHeader() (*eth.ExecutionPayloadHeader, error) {
	return w.b.ExecutionPayloadHeader, nil
}
