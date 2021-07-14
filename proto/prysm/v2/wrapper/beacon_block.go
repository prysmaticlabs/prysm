package wrapper

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
	"github.com/prysmaticlabs/prysm/shared/interfaces/version"
	"google.golang.org/protobuf/proto"
)

// AltairSignedBeaconBlock is a convenience wrapper around a altair beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type AltairSignedBeaconBlock struct {
	b *prysmv2.SignedBeaconBlock
}

// WrappedAltairSignedBeaconBlock is constructor which wraps a protobuf altair block
// with the block wrapper.
func WrappedAltairSignedBeaconBlock(b *prysmv2.SignedBeaconBlock) AltairSignedBeaconBlock {
	return AltairSignedBeaconBlock{b: b}
}

// Signature returns the respective block signature.
func (w AltairSignedBeaconBlock) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w AltairSignedBeaconBlock) Block() interfaces.BeaconBlock {
	return WrappedAltairBeaconBlock(w.b.Block)
}

// IsNil checks if the underlying beacon block is
// nil.
func (w AltairSignedBeaconBlock) IsNil() bool {
	return w.b == nil
}

// Copy performs a deep copy of the signed beacon block
// object.
func (w AltairSignedBeaconBlock) Copy() interfaces.SignedBeaconBlock {
	return WrappedAltairSignedBeaconBlock(copyutil.CopySignedBeaconBlockAltair(w.b))
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz
// form.
func (w AltairSignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block to its relevant ssz
// form to the provided byte buffer.
func (w AltairSignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized signed block
func (w AltairSignedBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the signed beacon block from its relevant ssz
// form.
func (w AltairSignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf
// interface.
func (w AltairSignedBeaconBlock) Proto() proto.Message {
	return w.b
}

// PbAltairBlock returns the underlying protobuf object.
func (w AltairSignedBeaconBlock) PbAltairBlock() (*prysmv2.SignedBeaconBlock, error) {
	return w.b, nil
}

// PbPhase0Block is a stub.
func (w AltairSignedBeaconBlock) PbPhase0Block() (*ethpb.SignedBeaconBlock, error) {
	return nil, errors.New("unsupported phase0 block")
}

// Version of the underlying protobuf object.
func (w AltairSignedBeaconBlock) Version() int {
	return version.Altair
}

// AltairBeaconBlock is the wrapper for the actual block.
type AltairBeaconBlock struct {
	b *prysmv2.BeaconBlock
}

// WrappedAltairBeaconBlock is constructor which wraps a protobuf altair object
// with the block wrapper.
func WrappedAltairBeaconBlock(b *prysmv2.BeaconBlock) AltairBeaconBlock {
	return AltairBeaconBlock{b: b}
}

// Slot returns the respective slot of the block.
func (w AltairBeaconBlock) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns proposer index of the beacon block.
func (w AltairBeaconBlock) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w AltairBeaconBlock) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w AltairBeaconBlock) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w AltairBeaconBlock) Body() interfaces.BeaconBlockBody {
	return WrappedAltairBeaconBlockBody(w.b.Body)
}

// IsNil checks if the beacon block is nil.
func (w AltairBeaconBlock) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block.
func (w AltairBeaconBlock) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w AltairBeaconBlock) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block to its relevant ssz
// form to the provided byte buffer.
func (w AltairBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of serialized block.
func (w AltairBeaconBlock) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshalls the beacon block from its relevant ssz
// form.
func (w AltairBeaconBlock) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w AltairBeaconBlock) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (w AltairBeaconBlock) Version() int {
	return version.Altair
}

// AltairBeaconBlockBody is a wrapper of a beacon block body.
type AltairBeaconBlockBody struct {
	b *prysmv2.BeaconBlockBody
}

// WrappedAltairBeaconBlockBody is constructor which wraps a protobuf altair object
// with the block wrapper.
func WrappedAltairBeaconBlockBody(b *prysmv2.BeaconBlockBody) AltairBeaconBlockBody {
	return AltairBeaconBlockBody{b: b}
}

// RandaoReveal returns the randao reveal from the block body.
func (w AltairBeaconBlockBody) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w AltairBeaconBlockBody) Eth1Data() *ethpb.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w AltairBeaconBlockBody) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w AltairBeaconBlockBody) ProposerSlashings() []*ethpb.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w AltairBeaconBlockBody) AttesterSlashings() []*ethpb.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w AltairBeaconBlockBody) Attestations() []*ethpb.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w AltairBeaconBlockBody) Deposits() []*ethpb.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w AltairBeaconBlockBody) VoluntaryExits() []*ethpb.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w AltairBeaconBlockBody) SyncAggregate() (*prysmv2.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w AltairBeaconBlockBody) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w AltairBeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w AltairBeaconBlockBody) Proto() proto.Message {
	return w.b
}
