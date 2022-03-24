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
	_ = block.SignedBeaconBlock(&Phase0SignedBeaconBlock{})
	_ = block.BeaconBlock(&Phase0BeaconBlock{})
	_ = block.BeaconBlockBody(&Phase0BeaconBlockBody{})
)

// Phase0SignedBeaconBlock is a convenience wrapper around a phase 0 beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type Phase0SignedBeaconBlock struct {
	b *eth.SignedBeaconBlock
}

// WrappedPhase0SignedBeaconBlock is constructor which wraps a protobuf phase 0 block
// with the block wrapper.
// Deprecated: use WrappedSignedBeaconBlock instead.
func WrappedPhase0SignedBeaconBlock(b *eth.SignedBeaconBlock) block.SignedBeaconBlock {
	return Phase0SignedBeaconBlock{b: b}
}

// Signature returns the respective block signature.
func (w Phase0SignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w Phase0SignedBeaconBlock) Block() block.BeaconBlock {
	return WrappedPhase0BeaconBlock(w.b.Block)
}

// IsNil checks if the underlying beacon block is
// nil.
func (w Phase0SignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.Block().IsNil()
}

// Copy performs a deep copy of the signed beacon block
// object.
func (w Phase0SignedBeaconBlock) Copy() block.SignedBeaconBlock {
	return WrappedPhase0SignedBeaconBlock(eth.CopySignedBeaconBlock(w.b))
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz
// form.
func (w Phase0SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w Phase0SignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w Phase0SignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w Phase0SignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf
// interface.
func (w Phase0SignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w Phase0SignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Phase0{Phase0: w.b},
	}, nil
}

// PbPhase0Block returns the underlying protobuf object.
func (w Phase0SignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return w.b, nil
}

// PbAltairBlock is a stub.
func (Phase0SignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, ErrUnsupportedAltairBlock
}

// PbBellatrixBlock is a stub.
func (Phase0SignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, ErrUnsupportedBellatrixBlock
}

// Version of the underlying protobuf object.
func (Phase0SignedBeaconBlock) Version() int {
	return version.Phase0
}

func (w Phase0SignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// Phase0BeaconBlock is the wrapper for the actual block.
type Phase0BeaconBlock struct {
	b *eth.BeaconBlock
}

// WrappedPhase0BeaconBlock is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
//
// Deprecated: Use WrappedBeaconBlock.
func WrappedPhase0BeaconBlock(b *eth.BeaconBlock) block.BeaconBlock {
	return Phase0BeaconBlock{b: b}
}

// Slot returns the respective slot of the block.
func (w Phase0BeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w Phase0BeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w Phase0BeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w Phase0BeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w Phase0BeaconBlock) Body() block.BeaconBlockBody {
	return WrappedPhase0BeaconBlockBody(w.b.Body)
}

// IsNil checks if the beacon block is nil.
func (w Phase0BeaconBlock) IsNil() bool {
	return w.b == nil || w.Body().IsNil()
}

// HashTreeRoot returns the ssz root of the block.
func (w Phase0BeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w Phase0BeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w Phase0BeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w Phase0BeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w Phase0BeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w Phase0BeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w Phase0BeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (Phase0BeaconBlock) Version() int {
	return version.Phase0
}

func (w Phase0BeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_Block{
		Block: w.b,
	}
}

// Phase0BeaconBlockBody is a wrapper of a beacon block body.
type Phase0BeaconBlockBody struct {
	b *eth.BeaconBlockBody
}

// WrappedPhase0BeaconBlockBody is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func WrappedPhase0BeaconBlockBody(b *eth.BeaconBlockBody) block.BeaconBlockBody {
	return Phase0BeaconBlockBody{b: b}
}

// RandaoReveal returns the randao reveal from the block body.
func (w Phase0BeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w Phase0BeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w Phase0BeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w Phase0BeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w Phase0BeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w Phase0BeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w Phase0BeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w Phase0BeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (Phase0BeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return nil, errors.New("Sync aggregate is not supported in phase 0 block")
}

// IsNil checks if the block body is nil.
func (w Phase0BeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w Phase0BeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w Phase0BeaconBlockBody) Proto() proto.Message {
	return w.b
}

// ExecutionPayload is a stub.
func (w Phase0BeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return nil, errors.Wrapf(ErrUnsupportedField, "ExecutionPayload for %T", w)
}
