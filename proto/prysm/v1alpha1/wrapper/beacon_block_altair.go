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
	_ = block.SignedBeaconBlock(&altairSignedBeaconBlock{})
	_ = block.BeaconBlock(&altairBeaconBlock{})
	_ = block.BeaconBlockBody(&altairBeaconBlockBody{})
)

// altairSignedBeaconBlock is a convenience wrapper around a altair beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type altairSignedBeaconBlock struct {
	b *eth.SignedBeaconBlockAltair
}

// WrappedAltairSignedBeaconBlock is constructor which wraps a protobuf altair block
// with the block wrapper.
// Deprecated: use WrappedSignedBeaconBlock instead.
func WrappedAltairSignedBeaconBlock(b *eth.SignedBeaconBlockAltair) (block.SignedBeaconBlock, error) {
	w := altairSignedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w altairSignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w altairSignedBeaconBlock) Block() block.BeaconBlock {
	return altairBeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is
// nil.
func (w altairSignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block
// object.
func (w altairSignedBeaconBlock) Copy() block.SignedBeaconBlock {
	return altairSignedBeaconBlock{b: eth.CopySignedBeaconBlockAltair(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz
// form.
func (w altairSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w altairSignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w altairSignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w altairSignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf
// interface.
func (w altairSignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w altairSignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Altair{Altair: w.b},
	}, nil
}

// PbAltairBlock returns the underlying protobuf object.
func (w altairSignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return w.b, nil
}

// PbPhase0Block is a stub.
func (altairSignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbBellatrixBlock is a stub.
func (altairSignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, ErrUnsupportedBellatrixBlock
}

// Version of the underlying protobuf object.
func (altairSignedBeaconBlock) Version() int {
	return version.Altair
}

func (w altairSignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// altairBeaconBlock is the wrapper for the actual block.
type altairBeaconBlock struct {
	b *eth.BeaconBlockAltair
}

// WrappedAltairBeaconBlock is constructor which wraps a protobuf altair object
// with the block wrapper.
//
// Deprecated: Use WrappedBeaconBlock.
func WrappedAltairBeaconBlock(b *eth.BeaconBlockAltair) (block.BeaconBlock, error) {
	w := altairBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w altairBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w altairBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w altairBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w altairBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w altairBeaconBlock) Body() block.BeaconBlockBody {
	return altairBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w altairBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w altairBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w altairBeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w altairBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w altairBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w altairBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w altairBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w altairBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (altairBeaconBlock) Version() int {
	return version.Altair
}

func (w altairBeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlockV2{
		BlockV2: w.b,
	}
}

// altairBeaconBlockBody is a wrapper of a beacon block body.
type altairBeaconBlockBody struct {
	b *eth.BeaconBlockBodyAltair
}

// WrappedAltairBeaconBlockBody is constructor which wraps a protobuf altair object
// with the block wrapper.
func WrappedAltairBeaconBlockBody(b *eth.BeaconBlockBodyAltair) (block.BeaconBlockBody, error) {
	w := altairBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w altairBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w altairBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w altairBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w altairBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w altairBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w altairBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w altairBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w altairBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w altairBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w altairBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w altairBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w altairBeaconBlockBody) Proto() proto.Message {
	return w.b
}

// ExecutionPayload is a stub.
func (w altairBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return nil, errors.Wrapf(ErrUnsupportedField, "%T", w)
}
