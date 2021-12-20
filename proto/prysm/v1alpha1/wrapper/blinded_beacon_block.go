package wrapper

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

var (
	_ = block.SignedBeaconBlock(&mergeSignedBlindedBeaconBlock{})
	_ = block.BeaconBlock(&mergeBlindedBeaconBlock{})
	_ = block.BeaconBlockBody(&mergeBlindedBeaconBlockBody{})
)

type mergeSignedBlindedBeaconBlock struct {
	b *eth.SignedBlindedBeaconBlockMerge
}

// WrappedMergeSignedBlindedBeaconBlock is constructor which wraps a protobuf merge block with the block wrapper.
func WrappedMergeSignedBlindedBeaconBlock(b *eth.SignedBlindedBeaconBlockMerge) (block.SignedBeaconBlock, error) {
	w := mergeSignedBlindedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w mergeSignedBlindedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w mergeSignedBlindedBeaconBlock) Block() block.BeaconBlock {
	return mergeBlindedBeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w mergeSignedBlindedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w mergeSignedBlindedBeaconBlock) Copy() block.SignedBeaconBlock {
	return mergeSignedBlindedBeaconBlock{b: eth.CopySignedBlindedBeaconBlockMerge(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w mergeSignedBlindedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w mergeSignedBlindedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w mergeSignedBlindedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w mergeSignedBlindedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w mergeSignedBlindedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbBlindedMergeBlock returns the underlying protobuf object.
func (w mergeSignedBlindedBeaconBlock) PbBlindedMergeBlock() (*eth.SignedBlindedBeaconBlockMerge, error) {
	return w.b, nil
}

// PbMergeBlock returns the underlying protobuf object.
func (w mergeSignedBlindedBeaconBlock) PbMergeBlock() (*eth.SignedBeaconBlockMerge, error) {
	return nil, errors.New("unsupported merge block")
}

// PbPhase0Block is a stub.
func (_ mergeSignedBlindedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (_ mergeSignedBlindedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, errors.New("unsupported altair block")
}

// Version of the underlying protobuf object.
func (_ mergeSignedBlindedBeaconBlock) Version() int {
	return version.Merge
}

func (w mergeSignedBlindedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// mergeBlindedBeaconBlock is the wrapper for the actual block.
type mergeBlindedBeaconBlock struct {
	b *eth.BlindedBeaconBlockMerge
}

// WrappedMergeBlindedBeaconBlock is constructor which wraps a protobuf merge object
// with the block wrapper.
func WrappedMergeBlindedBeaconBlock(b *eth.BlindedBeaconBlockMerge) (block.BeaconBlock, error) {
	w := mergeBlindedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w mergeBlindedBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w mergeBlindedBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w mergeBlindedBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w mergeBlindedBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w mergeBlindedBeaconBlock) Body() block.BeaconBlockBody {
	return mergeBlindedBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w mergeBlindedBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w mergeBlindedBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w mergeBlindedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w mergeBlindedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w mergeBlindedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w mergeBlindedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w mergeBlindedBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (_ mergeBlindedBeaconBlock) Version() int {
	return version.Merge
}

// mergeBlindedBeaconBlockBody is a wrapper of a beacon block body.
type mergeBlindedBeaconBlockBody struct {
	b *eth.BlindedBeaconBlockBodyMerge
}

// WrappedMergeBlindedBeaconBlockBody is constructor which wraps a protobuf merge object
// with the block wrapper.
func WrappedMergeBlindedBeaconBlockBody(b *eth.BlindedBeaconBlockBodyMerge) (block.BeaconBlockBody, error) {
	w := mergeBlindedBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w mergeBlindedBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w mergeBlindedBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w mergeBlindedBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w mergeBlindedBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w mergeBlindedBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w mergeBlindedBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w mergeBlindedBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w mergeBlindedBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w mergeBlindedBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w mergeBlindedBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w mergeBlindedBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w mergeBlindedBeaconBlockBody) Proto() proto.Message {
	return w.b
}

// ExecutionPayload returns the Execution payload of the block body.
func (w mergeBlindedBeaconBlockBody) ExecutionPayload() (*eth.ExecutionPayload, error) {
	return nil, errors.New("ExecutionPayload is not supported in mergeBlindedBeaconBlockBody")
}

// ExecutionPayloadHeader returns the Execution payload header of the block body.
func (w mergeBlindedBeaconBlockBody) ExecutionPayloadHeader() (*eth.ExecutionPayloadHeader, error) {
	return w.b.ExecutionPayloadHeader, nil
}
