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
	_ = interfaces.SignedBeaconBlock(&signedBlindedBeaconBlockBellatrix{})
	_ = interfaces.BeaconBlock(&blindedBeaconBlockBellatrix{})
	_ = interfaces.BeaconBlockBody(&blindedBeaconBlockBodyBellatrix{})
)

// signedBlindedBeaconBlockBellatrix is a convenience wrapper around a Bellatrix blinded beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type signedBlindedBeaconBlockBellatrix struct {
	b *eth.SignedBlindedBeaconBlockBellatrix
}

// wrappedBellatrixSignedBlindedBeaconBlock is a constructor which wraps a protobuf Bellatrix blinded block with the block wrapper.
func wrappedBellatrixSignedBlindedBeaconBlock(b *eth.SignedBlindedBeaconBlockBellatrix) (interfaces.SignedBeaconBlock, error) {
	w := signedBlindedBeaconBlockBellatrix{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Signature returns the respective block signature.
func (w signedBlindedBeaconBlockBellatrix) Signature() []byte {
	return w.b.Signature
}

// Block returns the underlying beacon block object.
func (w signedBlindedBeaconBlockBellatrix) Block() interfaces.BeaconBlock {
	return blindedBeaconBlockBellatrix{b: w.b.Block}
}

// IsNil checks if the underlying beacon block is nil.
func (w signedBlindedBeaconBlockBellatrix) IsNil() bool {
	return w.b == nil || w.b.Block == nil
}

// Copy performs a deep copy of the signed beacon block object.
func (w signedBlindedBeaconBlockBellatrix) Copy() interfaces.SignedBeaconBlock {
	return signedBlindedBeaconBlockBellatrix{b: eth.CopySignedBlindedBeaconBlockBellatrix(w.b)}
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (w signedBlindedBeaconBlockBellatrix) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (w signedBlindedBeaconBlockBellatrix) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (w signedBlindedBeaconBlockBellatrix) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
// form.
func (w signedBlindedBeaconBlockBellatrix) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the block in its underlying protobuf interface.
func (w signedBlindedBeaconBlockBellatrix) Proto() proto.Message {
	return w.b
}

// PbGenericBlock returns a generic signed beacon block.
func (w signedBlindedBeaconBlockBellatrix) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	return &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: w.b},
	}, nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (signedBlindedBeaconBlockBellatrix) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	return nil, ErrUnsupportedBellatrixBlock
}

// PbBlindedBellatrixBlock returns the underlying protobuf object.
func (w signedBlindedBeaconBlockBellatrix) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	return w.b, nil
}

// PbPhase0Block returns the underlying protobuf object.
func (signedBlindedBeaconBlockBellatrix) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	return nil, ErrUnsupportedPhase0Block
}

// PbAltairBlock returns the underlying protobuf object.
func (signedBlindedBeaconBlockBellatrix) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	return nil, ErrUnsupportedAltairBlock
}

// Version of the underlying protobuf object.
func (signedBlindedBeaconBlockBellatrix) Version() int {
	return version.BellatrixBlind
}

// Header converts the underlying protobuf object from blinded block to header format.
func (w signedBlindedBeaconBlockBellatrix) Header() (*eth.SignedBeaconBlockHeader, error) {
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

// blindedBeaconBlockBellatrix is the wrapper for the actual block.
type blindedBeaconBlockBellatrix struct {
	b *eth.BlindedBeaconBlockBellatrix
}

// wrappedBellatrixBlindedBeaconBlock is a constructor which wraps a protobuf Bellatrix object
// with the block wrapper.
func wrappedBellatrixBlindedBeaconBlock(b *eth.BlindedBeaconBlockBellatrix) (interfaces.BeaconBlock, error) {
	w := blindedBeaconBlockBellatrix{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Slot returns the respective slot of the block.
func (w blindedBeaconBlockBellatrix) Slot() types.Slot {
	return w.b.Slot
}

// ProposerIndex returns the proposer index of the beacon block.
func (w blindedBeaconBlockBellatrix) ProposerIndex() types.ValidatorIndex {
	return w.b.ProposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (w blindedBeaconBlockBellatrix) ParentRoot() []byte {
	return w.b.ParentRoot
}

// StateRoot returns the state root of the beacon block.
func (w blindedBeaconBlockBellatrix) StateRoot() []byte {
	return w.b.StateRoot
}

// Body returns the underlying block body.
func (w blindedBeaconBlockBellatrix) Body() interfaces.BeaconBlockBody {
	return blindedBeaconBlockBodyBellatrix{b: w.b.Body}
}

// IsNil checks if the beacon block is nil.
func (w blindedBeaconBlockBellatrix) IsNil() bool {
	return w.b == nil
}

// IsBlinded checks if the beacon block is a blinded block.
func (blindedBeaconBlockBellatrix) IsBlinded() bool {
	return true
}

// HashTreeRoot returns the ssz root of the block.
func (w blindedBeaconBlockBellatrix) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (w blindedBeaconBlockBellatrix) HashTreeRootWith(hh *ssz.Hasher) error {
	return w.b.HashTreeRootWith(hh)
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (w blindedBeaconBlockBellatrix) MarshalSSZ() ([]byte, error) {
	return w.b.MarshalSSZ()
}

// MarshalSSZTo marshals the beacon block's ssz
// form to the provided byte buffer.
func (w blindedBeaconBlockBellatrix) MarshalSSZTo(dst []byte) ([]byte, error) {
	return w.b.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized block.
func (w blindedBeaconBlockBellatrix) SizeSSZ() int {
	return w.b.SizeSSZ()
}

// UnmarshalSSZ unmarshals the beacon block from its relevant ssz
// form.
func (w blindedBeaconBlockBellatrix) UnmarshalSSZ(buf []byte) error {
	return w.b.UnmarshalSSZ(buf)
}

// Proto returns the underlying block object in its
// proto form.
func (w blindedBeaconBlockBellatrix) Proto() proto.Message {
	return w.b
}

// Version of the underlying protobuf object.
func (blindedBeaconBlockBellatrix) Version() int {
	return version.BellatrixBlind
}

// AsSignRequestObject returns the underlying sign request object.
func (w blindedBeaconBlockBellatrix) AsSignRequestObject() validatorpb.SignRequestObject {
	return &validatorpb.SignRequest_BlindedBlockV3{
		BlindedBlockV3: w.b,
	}
}

// blindedBeaconBlockBodyBellatrix is a wrapper of a beacon block body.
type blindedBeaconBlockBodyBellatrix struct {
	b *eth.BlindedBeaconBlockBodyBellatrix
}

// wrappedBellatrixBlindedBeaconBlockBody is a constructor which wraps a protobuf bellatrix object
// with the block wrapper.
func wrappedBellatrixBlindedBeaconBlockBody(b *eth.BlindedBeaconBlockBodyBellatrix) (interfaces.BeaconBlockBody, error) {
	w := blindedBeaconBlockBodyBellatrix{b: b}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// RandaoReveal returns the randao reveal from the block body.
func (w blindedBeaconBlockBodyBellatrix) RandaoReveal() []byte {
	return w.b.RandaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (w blindedBeaconBlockBodyBellatrix) Eth1Data() *eth.Eth1Data {
	return w.b.Eth1Data
}

// Graffiti returns the graffiti in the block.
func (w blindedBeaconBlockBodyBellatrix) Graffiti() []byte {
	return w.b.Graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (w blindedBeaconBlockBodyBellatrix) ProposerSlashings() []*eth.ProposerSlashing {
	return w.b.ProposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (w blindedBeaconBlockBodyBellatrix) AttesterSlashings() []*eth.AttesterSlashing {
	return w.b.AttesterSlashings
}

// Attestations returns the stored attestations in the block.
func (w blindedBeaconBlockBodyBellatrix) Attestations() []*eth.Attestation {
	return w.b.Attestations
}

// Deposits returns the stored deposits in the block.
func (w blindedBeaconBlockBodyBellatrix) Deposits() []*eth.Deposit {
	return w.b.Deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (w blindedBeaconBlockBodyBellatrix) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return w.b.VoluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (w blindedBeaconBlockBodyBellatrix) SyncAggregate() (*eth.SyncAggregate, error) {
	return w.b.SyncAggregate, nil
}

// IsNil checks if the block body is nil.
func (w blindedBeaconBlockBodyBellatrix) IsNil() bool {
	return w.b == nil
}

// HashTreeRoot returns the ssz root of the block body.
func (w blindedBeaconBlockBodyBellatrix) HashTreeRoot() ([32]byte, error) {
	return w.b.HashTreeRoot()
}

// Proto returns the underlying proto form of the block
// body.
func (w blindedBeaconBlockBodyBellatrix) Proto() proto.Message {
	return w.b
}

func (w blindedBeaconBlockBodyBellatrix) Execution() (interfaces.ExecutionData, error) {
	return WrappedExecutionPayloadHeader(w.b.ExecutionPayloadHeader)
}
