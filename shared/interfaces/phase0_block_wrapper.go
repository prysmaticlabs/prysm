package interfaces

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
	"google.golang.org/protobuf/proto"
)

// Phase0SignedBeaconBlock is a convenience wrapper around a phase 0 beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type Phase0SignedBeaconBlock struct {
	b *ethpb.SignedBeaconBlock
}

// WrappedPhase0SignedBeaconBlock is constructor which wraps a protobuf phase 0 block
// with the block wrapper.
func WrappedPhase0SignedBeaconBlock(b *ethpb.SignedBeaconBlock) Phase0SignedBeaconBlock {
	return Phase0SignedBeaconBlock{b: b}
}

// Signature returns the respective block signature.
func (w Phase0SignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w Phase0SignedBeaconBlock) Block() BeaconBlock {
	return WrappedPhase0BeaconBlock(w.b.Block)
}

// IsNil checks if the underlying beacon block is
// nil.
func (w Phase0SignedBeaconBlock) IsNil() bool {
	return w.b == nil || w.Block().IsNil()
}

// Copy performs a deep copy of the signed beacon block
// object.
func (w Phase0SignedBeaconBlock) Copy() SignedBeaconBlock {
	return WrappedPhase0SignedBeaconBlock(copyutil.CopySignedBeaconBlock(w.b))
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz
// form.
func (w Phase0SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// Proto returns the block in its underlying protobuf
// interface.
func (w Phase0SignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbPhase0Block returns the underlying protobuf object.
func (w Phase0SignedBeaconBlock) PbPhase0Block() (*ethpb.SignedBeaconBlock, error) {
	return w.b, nil
}

// Phase0BeaconBlock is the wrapper for the actual block.
type Phase0BeaconBlock struct {
	b *ethpb.BeaconBlock
}

// WrappedPhase0BeaconBlock is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func WrappedPhase0BeaconBlock(b *ethpb.BeaconBlock) Phase0BeaconBlock {
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
func (w Phase0BeaconBlock) Body() BeaconBlockBody {
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

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w Phase0BeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// Proto returns the underlying block object in its
// proto form.
func (w Phase0BeaconBlock) Proto() proto.Message {
	return w.b
}

// Phase0BeaconBlockBody is a wrapper of a beacon block body.
type Phase0BeaconBlockBody struct {
	b *ethpb.BeaconBlockBody
}

// WrappedPhase0BeaconBlockBody is constructor which wraps a protobuf phase 0 object
// with the block wrapper.
func WrappedPhase0BeaconBlockBody(b *ethpb.BeaconBlockBody) Phase0BeaconBlockBody {
	return Phase0BeaconBlockBody{b: b}
}

// RandaoReveal returns the randao reveal from the block body.
func (w Phase0BeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w Phase0BeaconBlockBody) Eth1Data() *ethpb.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w Phase0BeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w Phase0BeaconBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w Phase0BeaconBlockBody) AttesterSlashings() []*ethpb.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w Phase0BeaconBlockBody) Attestations() []*ethpb.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w Phase0BeaconBlockBody) Deposits() []*ethpb.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w Phase0BeaconBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit {
	return w.b.VoluntaryExits
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
