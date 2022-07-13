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
	_ = interfaces.SignedBeaconBlock(&Phase0SignedBeaconBlock{})
	_ = interfaces.BeaconBlock(&Phase0BeaconBlock{})
	_ = interfaces.BeaconBlockBody(&Phase0BeaconBlockBody{})
)

// Phase0SignedBeaconBlock is a convenience wrapper around a phase 0 beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type Phase0SignedBeaconBlock struct {
	b *eth.SignedBeaconBlock
}

// wrappedPhase0SignedBeaconBlock is constructor which wraps a protobuf phase 0 block
// with the block wrapper.
func wrappedPhase0SignedBeaconBlock(b *eth.SignedBeaconBlock) interfaces.SignedBeaconBlock {
	return Phase0SignedBeaconBlock{b: b}
}

// Signature returns the respective block signature.
func (w Phase0SignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w Phase0SignedBeaconBlock) Block() interfaces.BeaconBlock {
	return wrappedPhase0BeaconBlock(w.b.Block)
}

// IsNil checks if the underlying beacon block is
// nil.
func (w Phase0SignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.Block().IsNil()
}

// Copy performs a deep copy of the signed beacon block
// object.
func (w Phase0SignedBeaconBlock) Copy() interfaces.SignedBeaconBlock {
	return wrappedPhase0SignedBeaconBlock(eth.CopySignedBeaconBlock(w.b))
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz
// form.
func (w Phase0SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (w Phase0SignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (w Phase0SignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
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

// PbBlindedBellatrixBlock is a stub.
func (Phase0SignedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	return nil, ErrUnsupportedBlindedBellatrixBlock
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

// wrappedPhase0BeaconBlock is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func wrappedPhase0BeaconBlock(b *eth.BeaconBlock) interfaces.BeaconBlock {
	return Phase0BeaconBlock{b: b}
}

// Slot returns the respective slot of the block.
func (w Phase0BeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns the proposer index of the beacon block.
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
func (w Phase0BeaconBlock) Body() interfaces.BeaconBlockBody {
	return wrappedPhase0BeaconBlockBody(w.b.Body)
}

// IsNil checks if the beacon block is nil.
func (w Phase0BeaconBlock) IsNil() bool {
	return w.b == nil || w.Body().IsNil()
}

// IsBlinded checks if the beacon block is a blinded block.
func (Phase0BeaconBlock) IsBlinded() bool {
	return false
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

// MarshalSSZTo marshals the beacon block's ssz
// form to the provided byte buffer.
func (w Phase0BeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized block.
func (w Phase0BeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the beacon block from its relevant ssz
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

// AsSignRequestObject returns the underlying sign request object.
func (w Phase0BeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_Block{
		Block: w.b,
	}
}

// Phase0BeaconBlockBody is a wrapper of a beacon block body.
type Phase0BeaconBlockBody struct {
	b *eth.BeaconBlockBody
}

// wrappedPhase0BeaconBlockBody is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func wrappedPhase0BeaconBlockBody(b *eth.BeaconBlockBody) interfaces.BeaconBlockBody {
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

// Execution is a stub.
func (w Phase0BeaconBlockBody) Execution() (interfaces.ExecutionData, error) {
	return nil, errors.Wrapf(ErrUnsupportedField, "ExecutionPayload for %T", w)
}
