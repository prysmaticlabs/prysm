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

// miniDankShardingSignedBeaconBlock is a convenience wrapper around a miniDankSharding beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type miniDankShardingSignedBeaconBlock struct {
	b *eth.SignedBeaconBlockWithBlobKZGs
}

// WrappedMiniDankShardingSignedBeaconBlock is constructor which wraps a protobuf miniDankSharding block with the block wrapper.
func WrappedMiniDankShardingSignedBeaconBlock(b *eth.SignedBeaconBlockWithBlobKZGs) (block.SignedBeaconBlock, error) {
	w := miniDankShardingSignedBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w miniDankShardingSignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w miniDankShardingSignedBeaconBlock) Block() block.BeaconBlock {
	return miniDankShardingBeaconBlock{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w miniDankShardingSignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w miniDankShardingSignedBeaconBlock) Copy() block.SignedBeaconBlock {
	return miniDankShardingSignedBeaconBlock{b: w.b} // TODO: Add copy method
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w miniDankShardingSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w miniDankShardingSignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w miniDankShardingSignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w miniDankShardingSignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w miniDankShardingSignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbminiDankShardingBlock returns the underlying protobuf object.
func (w miniDankShardingSignedBeaconBlock) PbMiniDankShardingBlock() (*eth.SignedBeaconBlockWithBlobKZGs, error) {
	return w.b, nil
}

// PbPhase0Block is a stub.
func (_ miniDankShardingSignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (_ miniDankShardingSignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, errors.New("unsupported altair block")
}

// PbminiDankShardingBlock returns the underlying protobuf object.
func (w miniDankShardingSignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, errors.New("unsupported bellatrix block")
}

// Version of the underlying protobuf object.
func (_ miniDankShardingSignedBeaconBlock) Version() int {
	return version.MiniDankSharding
}

func (w miniDankShardingSignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// miniDankShardingBeaconBlock is the wrapper for the actual block.
type miniDankShardingBeaconBlock struct {
	b *eth.BeaconBlockWithBlobKZGs
}

// WrappedMiniDankShardingBeaconBlock is constructor which wraps a protobuf miniDankSharding object
// with the block wrapper.
func WrappedMiniDankShardingBeaconBlock(b *eth.BeaconBlockWithBlobKZGs) (block.BeaconBlock, error) {
	w := miniDankShardingBeaconBlock{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w miniDankShardingBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w miniDankShardingBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w miniDankShardingBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w miniDankShardingBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w miniDankShardingBeaconBlock) Body() block.BeaconBlockBody {
	return miniDankShardingBeaconBlockBody{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w miniDankShardingBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w miniDankShardingBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w miniDankShardingBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w miniDankShardingBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w miniDankShardingBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w miniDankShardingBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w miniDankShardingBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (_ miniDankShardingBeaconBlock) Version() int {
	return version.MiniDankSharding
}

// miniDankShardingBeaconBlockBody is a wrapper of a beacon block body.
type miniDankShardingBeaconBlockBody struct {
	b *eth.BeaconBlockBodyWithBlobKZGs
}

// WrappedminiDankShardingBeaconBlockBody is constructor which wraps a protobuf miniDankSharding object
// with the block wrapper.
func WrappedMiniDankShardingBeaconBlockBody(b *eth.BeaconBlockBodyWithBlobKZGs) (block.BeaconBlockBody, error) {
	w := miniDankShardingBeaconBlockBody{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w miniDankShardingBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w miniDankShardingBeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w miniDankShardingBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w miniDankShardingBeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w miniDankShardingBeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w miniDankShardingBeaconBlockBody) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w miniDankShardingBeaconBlockBody) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w miniDankShardingBeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w miniDankShardingBeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w miniDankShardingBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w miniDankShardingBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w miniDankShardingBeaconBlockBody) Proto() proto.Message {
	return w.b
}

func (w miniDankShardingBeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	return w.b.ExecutionPayload, nil
}
