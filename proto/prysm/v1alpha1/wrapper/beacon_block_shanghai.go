package wrapper

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

// shanghaiSignedBeaconBlock is a convenience wrapper around a shanghai beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type shanghaiSignedBeaconBlock struct {
	b *eth.SignedBeaconBlockAndBlobs
}

// WrappedShanghaiSignedBeaconBlock is constructor which wraps a protobuf shanghai block with the block wrapper.
func WrappedShanghaiSignedBeaconBlock(b *eth.SignedBeaconBlockAndBlobs) (block.SignedBeaconBlock, error) {
	w := shanghaiSignedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w shanghaiSignedBeaconBlock) Signature() []byte {
	return w.b.Block.Signature
}

// Block returns the underlying beacon block object.
func (w shanghaiSignedBeaconBlock) Block() block.BeaconBlock {
	return shanghaiBeaconBlock{b: w.b.Block.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w shanghaiSignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil || w.b.Block.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w shanghaiSignedBeaconBlock) Copy() block.SignedBeaconBlock {
	return shanghaiSignedBeaconBlock{b: w.b} // TODO: Add copy method
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w shanghaiSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w shanghaiSignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w shanghaiSignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w shanghaiSignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w shanghaiSignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbshanghaiBlock returns the underlying protobuf object.
func (w shanghaiSignedBeaconBlock) PbShanghaiBlock() (*eth.SignedBeaconBlockAndBlobs, error) {
	return w.b, nil
}

// PbPhase0Block is a stub.
func (_ shanghaiSignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (_ shanghaiSignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, errors.New("unsupported altair block")
}

// PbshanghaiBlock returns the underlying protobuf object.
func (w shanghaiSignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, errors.New("unsupported bellatrix block")
}

// Version of the underlying protobuf object.
func (_ shanghaiSignedBeaconBlock) Version() int {
	return version.Shanghai
}

func (w shanghaiSignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
	root, err := w.b.Block.Block.Body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not hash block")
	}

	return &eth.SignedBeaconBlockHeader{
		Header: &eth.BeaconBlockHeader{
			Slot:          w.b.Block.Block.Slot,
			ProposerIndex: w.b.Block.Block.ProposerIndex,
			ParentRoot:    w.b.Block.Block.ParentRoot,
			StateRoot:     w.b.Block.Block.StateRoot,
			BodyRoot:      root[:],
		},
		Signature: w.Signature(),
	}, nil
}

// shanghaiBeaconBlock is the wrapper for the actual block.
type shanghaiBeaconBlock struct {
	b *eth.BeaconBlockWithBlobKZGs
}

// WrappedShanghaiBeaconBlock is constructor which wraps a protobuf shanghai object
// with the block wrapper.
func WrappedShanghaiBeaconBlock(b *eth.BeaconBlockWithBlobKZGs) (block.BeaconBlock, error) {
	w := shanghaiBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w shanghaiBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w shanghaiBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w shanghaiBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w shanghaiBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w shanghaiBeaconBlock) Body() block.BeaconBlockBody {
	return shanghaiBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w shanghaiBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w shanghaiBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w shanghaiBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w shanghaiBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w shanghaiBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w shanghaiBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w shanghaiBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (_ shanghaiBeaconBlock) Version() int {
	return version.Shanghai
}

// shanghaiBeaconBlockBody is a wrapper of a beacon block body.
type shanghaiBeaconBlockBody struct {
	b *eth.BeaconBlockBodyWithBlobKZGs
}

// WrappedshanghaiBeaconBlockBody is constructor which wraps a protobuf shanghai object
// with the block wrapper.
func WrappedShanghaiBeaconBlockBody(b *eth.BeaconBlockBodyWithBlobKZGs) (block.BeaconBlockBody, error) {
	w := shanghaiBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w shanghaiBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w shanghaiBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w shanghaiBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w shanghaiBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w shanghaiBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w shanghaiBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w shanghaiBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w shanghaiBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w shanghaiBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w shanghaiBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w shanghaiBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w shanghaiBeaconBlockBody) Proto() proto.Message {
	return w.b
}

func (w shanghaiBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return w.b.ExecutionPayload, nil
}
