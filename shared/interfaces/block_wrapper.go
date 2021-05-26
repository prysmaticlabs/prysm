package interfaces

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"google.golang.org/protobuf/proto"
)

// WrappedSignedBeaconBlock is a convenience wrapper around a phase 0 beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type WrappedSignedBeaconBlock struct {
	b *ethpb.SignedBeaconBlock
}

// NewWrappedSignedBeaconBlock is constructor which wraps a protobuf phase 0 block
// with the block wrapper.
func NewWrappedSignedBeaconBlock(b *ethpb.SignedBeaconBlock) WrappedSignedBeaconBlock {
	return WrappedSignedBeaconBlock{b: b}
}

// Signature returns the respective block signature.
func (w WrappedSignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w WrappedSignedBeaconBlock) Block() BeaconBlock {
	return NewWrappedBeaconBlock(w.b.Block)
}

// IsNil checks if the underlying beacon block is
// nil.
func (w WrappedSignedBeaconBlock) IsNil() bool {
	return w.b == nil
}

// Copy performs a deep copy of the signed beacon block
// object.
func (w WrappedSignedBeaconBlock) Copy() WrappedSignedBeaconBlock {
	return NewWrappedSignedBeaconBlock(blockutil.CopySignedBeaconBlock(w.b))
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz
// form.
func (w WrappedSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// Proto returns the block in its underlying protobuf
// interface.
func (w WrappedSignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbPhase0Block returns the underlying protobuf object.
func (w WrappedSignedBeaconBlock) PbPhase0Block() (*ethpb.SignedBeaconBlock, error) {
	return w.b, nil
}

// WrappedBeaconBlock is the wrapper for the actual block.
type WrappedBeaconBlock struct {
	b *ethpb.BeaconBlock
}

// NewWrappedBeaconBlock is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func NewWrappedBeaconBlock(b *ethpb.BeaconBlock) WrappedBeaconBlock {
	return WrappedBeaconBlock{b: b}
}

// Slot returns the respective slot of the block.
func (w WrappedBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w WrappedBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w WrappedBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w WrappedBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w WrappedBeaconBlock) Body() BeaconBlockBody {
	return NewWrappedBeaconBlockBody(w.b.Body)
}

// IsNil checks if the beacon block is nil.
func (w WrappedBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w WrappedBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w WrappedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// Proto returns the underlying block object in its
// proto form.
func (w WrappedBeaconBlock) Proto() proto.Message {
	return w.b
}

// WrappedBeaconBlockBody is a wrapper of a beacon block body.
type WrappedBeaconBlockBody struct {
	b *ethpb.BeaconBlockBody
}

// NewWrappedBeaconBlockBody is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func NewWrappedBeaconBlockBody(b *ethpb.BeaconBlockBody) WrappedBeaconBlockBody {
	return WrappedBeaconBlockBody{b: b}
}

// RandaoReveal returns the randao reveal from the block body.
func (w WrappedBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w WrappedBeaconBlockBody) Eth1Data() *ethpb.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w WrappedBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w WrappedBeaconBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w WrappedBeaconBlockBody) AttesterSlashings() []*ethpb.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w WrappedBeaconBlockBody) Attestations() []*ethpb.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w WrappedBeaconBlockBody) Deposits() []*ethpb.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w WrappedBeaconBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// IsNIl checks if the block body is nil.
func (w WrappedBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w WrappedBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w WrappedBeaconBlockBody) Proto() proto.Message {
	return w.b
}
