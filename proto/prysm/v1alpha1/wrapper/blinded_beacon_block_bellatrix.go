package wrapper

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

var (
	_ = block.SignedBeaconBlock(&bellatrixSignedBlindedBeaconBlock{})
	_ = block.BeaconBlock(&bellatrixBlindedBeaconBlock{})
	_ = block.BeaconBlockBody(&bellatrixBlindedBeaconBlockBody{})
)

// bellatrixSignedBlindedBeaconBlock is a convenience wrapper around a Bellatrix beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type bellatrixSignedBlindedBeaconBlock struct {
	b *eth.SignedBlindedBeaconBlockBellatrix
}

// wrappedBellatrixSignedBlindedBeaconBlock is constructor which wraps a protobuf Bellatrix block with the block wrapper.
func wrappedBellatrixSignedBlindedBeaconBlock(b *eth.SignedBlindedBeaconBlockBellatrix) (block.SignedBeaconBlock, error) {
	w := bellatrixSignedBlindedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w bellatrixSignedBlindedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w bellatrixSignedBlindedBeaconBlock) Block() block.BeaconBlock {
	return bellatrixBlindedBeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w bellatrixSignedBlindedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w bellatrixSignedBlindedBeaconBlock) Copy() block.SignedBeaconBlock {
	return bellatrixSignedBlindedBeaconBlock{b: eth.CopySignedBlindedBeaconBlockBellatrix(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w bellatrixSignedBlindedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w bellatrixSignedBlindedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w bellatrixSignedBlindedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w bellatrixSignedBlindedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w bellatrixSignedBlindedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w bellatrixSignedBlindedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: w.b},
	}, nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (bellatrixSignedBlindedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, ErrUnsupportedBellatrixBlock
}

// PbBlindedBellatrixBlock is a stub.
func (w bellatrixSignedBlindedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	return w.b, nil
}

// PbPhase0Block is a stub.
func (bellatrixSignedBlindedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (bellatrixSignedBlindedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, ErrUnsupportedAltairBlock
}

// Version of the underlying protobuf object.
func (bellatrixSignedBlindedBeaconBlock) Version() int {
	return version.Bellatrix
}

func (w bellatrixSignedBlindedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// bellatrixBlindedBeaconBlock is the wrapper for the actual block.
type bellatrixBlindedBeaconBlock struct {
	b *eth.BlindedBeaconBlockBellatrix
}

// WrappedBellatrixBlindedBeaconBlock is constructor which wraps a protobuf Bellatrix object
// with the block wrapper.
//
// Deprecated: Use WrappedBeaconBlock.
func WrappedBellatrixBlindedBeaconBlock(b *eth.BlindedBeaconBlockBellatrix) (block.BeaconBlock, error) {
	w := bellatrixBlindedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w bellatrixBlindedBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w bellatrixBlindedBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w bellatrixBlindedBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w bellatrixBlindedBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w bellatrixBlindedBeaconBlock) Body() block.BeaconBlockBody {
	return bellatrixBlindedBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w bellatrixBlindedBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w bellatrixBlindedBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w bellatrixBlindedBeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w bellatrixBlindedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w bellatrixBlindedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w bellatrixBlindedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w bellatrixBlindedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w bellatrixBlindedBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (bellatrixBlindedBeaconBlock) Version() int {
	return version.Bellatrix
}

func (w bellatrixBlindedBeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlindedBlockV3{
		BlindedBlockV3: w.b,
	}
}

// bellatrixBlindedBeaconBlockBody is a wrapper of a beacon block body.
type bellatrixBlindedBeaconBlockBody struct {
	b *eth.BlindedBeaconBlockBodyBellatrix
}

// WrappedBellatrixBlindedBeaconBlockBody is constructor which wraps a protobuf bellatrix object
// with the block wrapper.
func WrappedBellatrixBlindedBeaconBlockBody(b *eth.BlindedBeaconBlockBodyBellatrix) (block.BeaconBlockBody, error) {
	w := bellatrixBlindedBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w bellatrixBlindedBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w bellatrixBlindedBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w bellatrixBlindedBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w bellatrixBlindedBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w bellatrixBlindedBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w bellatrixBlindedBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w bellatrixBlindedBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w bellatrixBlindedBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w bellatrixBlindedBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w bellatrixBlindedBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w bellatrixBlindedBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w bellatrixBlindedBeaconBlockBody) Proto() proto.Message {
	return w.b
}

// ExecutionPayload returns the Execution payload of the block body.
func (w bellatrixBlindedBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return nil, errors.Wrapf(ErrUnsupportedField, "ExecutionPayload for %T", w)
}

// ExecutionPayloadHeader is a stub.
func (w bellatrixBlindedBeaconBlockBody) ExecutionPayloadHeader() (*eth.ExecutionPayloadHeader, error) {
	return w.b.ExecutionPayloadHeader, nil
}
