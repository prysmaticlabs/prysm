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

// eip4844SignedBeaconBlock is a convenience wrapper around an eip4844 beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type eip4844SignedBeaconBlock struct {
	b *eth.SignedBeaconBlockWithBlobKZGs
}

// WrappedEip4844SignedBeaconBlock is constructor which wraps a protobuf eip4844 block with the block wrapper.
func WrappedEip4844SignedBeaconBlock(b *eth.SignedBeaconBlockWithBlobKZGs) (block.SignedBeaconBlock, error) {
	w := eip4844SignedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w eip4844SignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w eip4844SignedBeaconBlock) Block() block.BeaconBlock {
	return eip4844BeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w eip4844SignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w eip4844SignedBeaconBlock) Copy() block.SignedBeaconBlock {
	return eip4844SignedBeaconBlock{b: w.b} // TODO: Add copy method
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w eip4844SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w eip4844SignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w eip4844SignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w eip4844SignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w eip4844SignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w eip4844SignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Eip4844{Eip4844: w.b},
	}, nil
}

// PbEip4844Block returns the underlying protobuf object.
func (w eip4844SignedBeaconBlock) PbEip4844Block() (*eth.SignedBeaconBlockWithBlobKZGs, error) {
	return w.b, nil
}

// PbPhase0Block is a stub.
func (_ eip4844SignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (_ eip4844SignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, errors.New("unsupported altair block")
}

// Pbeip4844Block returns the underlying protobuf object.
func (w eip4844SignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, errors.New("unsupported bellatrix block")
}

// Version of the underlying protobuf object.
func (_ eip4844SignedBeaconBlock) Version() int {
	return version.EIP4844
}

func (w eip4844SignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// eip4844BeaconBlock is the wrapper for the actual block.
type eip4844BeaconBlock struct {
	b *eth.BeaconBlockWithBlobKZGs
}

// WrappedEip4844BeaconBlock is constructor which wraps a protobuf eip4844 object
// with the block wrapper.
func WrappedEip4844BeaconBlock(b *eth.BeaconBlockWithBlobKZGs) (block.BeaconBlock, error) {
	w := eip4844BeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w eip4844BeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w eip4844BeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w eip4844BeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w eip4844BeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w eip4844BeaconBlock) Body() block.BeaconBlockBody {
	return eip4844BeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w eip4844BeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w eip4844BeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w eip4844BeaconBlock) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w eip4844BeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w eip4844BeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w eip4844BeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w eip4844BeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w eip4844BeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (_ eip4844BeaconBlock) Version() int {
	return version.EIP4844
}

func (w eip4844BeaconBlock) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlockV4{
		BlockV4: w.b,
	}
}

// eip4844BeaconBlockBody is a wrapper of a beacon block body.
type eip4844BeaconBlockBody struct {
	b *eth.BeaconBlockBodyWithBlobKZGs
}

// Wrappedeip4844BeaconBlockBody is constructor which wraps a protobuf eip4844 object
// with the block wrapper.
func WrappedEip4844BeaconBlockBody(b *eth.BeaconBlockBodyWithBlobKZGs) (block.BeaconBlockBody, error) {
	w := eip4844BeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w eip4844BeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w eip4844BeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w eip4844BeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w eip4844BeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w eip4844BeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w eip4844BeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w eip4844BeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w eip4844BeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w eip4844BeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w eip4844BeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w eip4844BeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w eip4844BeaconBlockBody) Proto() proto.Message {
	return w.b
}

func (w eip4844BeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return w.b.ExecutionPayload, nil
}
