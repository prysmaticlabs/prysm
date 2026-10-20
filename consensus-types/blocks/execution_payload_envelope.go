package blocks

import (
	"bytes"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	field_params "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	consensus_types "github.com/OffchainLabs/prysm/v7/consensus-types"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type signedExecutionPayloadEnvelope struct {
	s *ethpb.SignedExecutionPayloadEnvelope
}

type executionPayloadEnvelope struct {
	p *ethpb.ExecutionPayloadEnvelope
}

// WrappedROSignedExecutionPayloadEnvelope wraps a Gloas signed execution payload envelope
// proto in a read-only interface. The Gloas container carries its beacon block root inline.
func WrappedROSignedExecutionPayloadEnvelope(s *ethpb.SignedExecutionPayloadEnvelope) (interfaces.ROSignedExecutionPayloadEnvelope, error) {
	w := signedExecutionPayloadEnvelope{s: s}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// WrappedROExecutionPayloadEnvelope wraps a Gloas execution payload envelope proto in a
// read-only interface.
func WrappedROExecutionPayloadEnvelope(p *ethpb.ExecutionPayloadEnvelope) (interfaces.ROExecutionPayloadEnvelope, error) {
	w := &executionPayloadEnvelope{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// Envelope returns the execution payload envelope as a read-only interface.
func (s signedExecutionPayloadEnvelope) Envelope() (interfaces.ROExecutionPayloadEnvelope, error) {
	return WrappedROExecutionPayloadEnvelope(s.s.Message)
}

// Version returns the fork the envelope container belongs to.
func (s signedExecutionPayloadEnvelope) Version() int {
	return version.Gloas
}

// Signature returns the BLS signature as a 96-byte array.
func (s signedExecutionPayloadEnvelope) Signature() [field_params.BLSSignatureLength]byte {
	return [field_params.BLSSignatureLength]byte(s.s.Signature)
}

// IsNil reports whether the signed envelope or its contents are invalid.
func (s signedExecutionPayloadEnvelope) IsNil() bool {
	if s.s == nil {
		return true
	}
	if len(s.s.Signature) != field_params.BLSSignatureLength {
		return true
	}
	if s.s.Message == nil {
		return true
	}
	if len(s.s.Message.BeaconBlockRoot) != field_params.RootLength {
		return true
	}
	if len(s.s.Message.ParentBeaconBlockRoot) != field_params.RootLength {
		return true
	}
	if s.s.Message.ExecutionRequests == nil {
		return true
	}
	if s.s.Message.Payload == nil {
		return true
	}
	w := executionPayloadEnvelope{p: s.s.Message}
	return w.IsNil()
}

// SigningRoot computes the signing root for the signed envelope with the provided domain.
func (s signedExecutionPayloadEnvelope) SigningRoot(domain []byte) (root [32]byte, err error) {
	return signing.ComputeSigningRoot(s.s.Message, domain)
}

// Proto returns the underlying protobuf message.
func (s signedExecutionPayloadEnvelope) Proto() proto.Message {
	return s.s
}

// IsNil reports whether the envelope or its required fields are invalid.
func (p *executionPayloadEnvelope) IsNil() bool {
	if p.p == nil {
		return true
	}
	if p.p.Payload == nil {
		return true
	}
	if len(p.p.BeaconBlockRoot) != field_params.RootLength {
		return true
	}
	if len(p.p.ParentBeaconBlockRoot) != field_params.RootLength {
		return true
	}
	return false
}

// IsBlinded reports whether the envelope contains a blinded payload.
func (p *executionPayloadEnvelope) IsBlinded() bool {
	return false
}

// Execution returns the execution payload as a read-only interface.
func (p *executionPayloadEnvelope) Execution() (interfaces.ExecutionData, error) {
	return WrappedExecutionPayloadGloas(p.p.Payload)
}

// ExecutionRequests returns the execution requests attached to the envelope.
func (p *executionPayloadEnvelope) ExecutionRequests() *enginev1.ExecutionRequestsGloas {
	return ethpb.CopyExecutionRequestsGloas(p.p.ExecutionRequests)
}

// BuilderIndex returns the proposer/builder index for the envelope.
func (p *executionPayloadEnvelope) BuilderIndex() primitives.BuilderIndex {
	return p.p.BuilderIndex
}

// BeaconBlockRoot returns the beacon block root referenced by the envelope.
func (p *executionPayloadEnvelope) BeaconBlockRoot() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.BeaconBlockRoot)
}

// Version returns the fork the envelope container belongs to.
func (p *executionPayloadEnvelope) Version() int {
	return version.Gloas
}

// ParentBeaconBlockRoot returns the parent beacon block root referenced by the envelope.
func (p *executionPayloadEnvelope) ParentBeaconBlockRoot() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.ParentBeaconBlockRoot)
}

// Slot returns the slot derived from the payload's slot_number field.
func (p *executionPayloadEnvelope) Slot() primitives.Slot {
	return primitives.Slot(p.p.Payload.SlotNumber)
}

// BlockHash returns the block hash from the execution payload.
func (p *executionPayloadEnvelope) BlockHash() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.Payload.BlockHash)
}

type blindedExecutionPayloadEnvelope struct {
	p *ethpb.BlindedExecutionPayloadEnvelope
}

// WrappedROBlindedExecutionPayloadEnvelope wraps a blinded execution payload envelope proto in a read-only interface.
func WrappedROBlindedExecutionPayloadEnvelope(p *ethpb.BlindedExecutionPayloadEnvelope) (interfaces.ROBlindedExecutionPayloadEnvelope, error) {
	w := &blindedExecutionPayloadEnvelope{p: p}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

func (p *blindedExecutionPayloadEnvelope) IsNil() bool {
	if p.p == nil {
		return true
	}
	if len(p.p.BeaconBlockRoot) != field_params.RootLength {
		return true
	}
	if len(p.p.ParentBeaconBlockRoot) != field_params.RootLength {
		return true
	}
	if len(p.p.BlockHash) != field_params.RootLength {
		return true
	}
	return false
}

func (p *blindedExecutionPayloadEnvelope) IsBlinded() bool {
	return true
}

func (p *blindedExecutionPayloadEnvelope) ExecutionRequests() *enginev1.ExecutionRequestsGloas {
	return ethpb.CopyExecutionRequestsGloas(p.p.ExecutionRequests)
}

func (p *blindedExecutionPayloadEnvelope) BuilderIndex() primitives.BuilderIndex {
	return p.p.BuilderIndex
}

func (p *blindedExecutionPayloadEnvelope) BeaconBlockRoot() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.BeaconBlockRoot)
}

func (p *blindedExecutionPayloadEnvelope) ParentBeaconBlockRoot() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.ParentBeaconBlockRoot)
}

func (p *blindedExecutionPayloadEnvelope) Slot() primitives.Slot {
	return p.p.Slot
}

func (p *blindedExecutionPayloadEnvelope) BlockHash() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.BlockHash)
}

// Parent slot fullness is committed by the child bid, not recorded in the parent itself.
func BlockBuiltOnParentPayload(parent, child interfaces.ReadOnlyBeaconBlock) (bool, error) {
	parentSigned, err := parent.Body().SignedExecutionPayloadBid()
	if err != nil {
		return false, err
	}
	childSigned, err := child.Body().SignedExecutionPayloadBid()
	if err != nil {
		return false, err
	}
	parentBid, err := parentSigned.Bid()
	if err != nil {
		return false, errors.Wrap(err, "parent execution payload bid")
	}
	childBid, err := childSigned.Bid()
	if err != nil {
		return false, errors.Wrap(err, "child execution payload bid")
	}
	parentHash := parentBid.BlockHash()
	childParentHash := childBid.ParentBlockHash()
	return childParentHash == parentHash, nil
}

// BlockBuiltOnParentEnvelope reports whether env is the payload of blk's parent block and blk builds on it.
// The root check matters when the parent was empty: an ancestor's payload then matches blk's parent hash.
// EIP-8411: wrappers built from by-range responses carry a zero root (the wire envelope has no root
// field); for those, binding falls back to the block hash check alone.
func BlockBuiltOnParentEnvelope(env interfaces.ROSignedExecutionPayloadEnvelope, blk ROBlock) (bool, error) {
	msg, err := env.Envelope()
	if err != nil {
		return false, err
	}
	if root := msg.BeaconBlockRoot(); root != ([field_params.RootLength]byte{}) && root != blk.Block().ParentRoot() {
		return false, nil
	}
	ex, err := msg.Execution()
	if err != nil {
		return false, err
	}
	ph, err := blk.ParentHash()
	if err != nil {
		return false, err
	}
	return bytes.Equal(ex.BlockHash(), ph[:]), nil
}

// ----------------------------------------------------------------------------
// Heze (EIP-8411) envelope wrappers: the container carries no beacon block
// root, so the binding is attached out of band at wrap time.
// ----------------------------------------------------------------------------

type signedExecutionPayloadEnvelopeHeze struct {
	s    *ethpb.SignedExecutionPayloadEnvelopeHeze
	root [field_params.RootLength]byte
}

type executionPayloadEnvelopeHeze struct {
	p    *ethpb.ExecutionPayloadEnvelopeHeze
	root [field_params.RootLength]byte
}

// WrappedROSignedExecutionPayloadEnvelopeHeze wraps a Heze signed envelope proto, bound to
// the given beacon block root (zero when the owning block is not yet known).
func WrappedROSignedExecutionPayloadEnvelopeHeze(s *ethpb.SignedExecutionPayloadEnvelopeHeze, beaconBlockRoot [field_params.RootLength]byte) (interfaces.ROSignedExecutionPayloadEnvelope, error) {
	w := signedExecutionPayloadEnvelopeHeze{s: s, root: beaconBlockRoot}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

// WrappedROExecutionPayloadEnvelopeHeze wraps a Heze envelope proto, bound to the given root.
func WrappedROExecutionPayloadEnvelopeHeze(p *ethpb.ExecutionPayloadEnvelopeHeze, beaconBlockRoot [field_params.RootLength]byte) (interfaces.ROExecutionPayloadEnvelope, error) {
	w := &executionPayloadEnvelopeHeze{p: p, root: beaconBlockRoot}
	if w.IsNil() {
		return nil, consensus_types.ErrNilObjectWrapped
	}
	return w, nil
}

func (s signedExecutionPayloadEnvelopeHeze) Envelope() (interfaces.ROExecutionPayloadEnvelope, error) {
	return WrappedROExecutionPayloadEnvelopeHeze(s.s.Message, s.root)
}

func (s signedExecutionPayloadEnvelopeHeze) Signature() [field_params.BLSSignatureLength]byte {
	return [field_params.BLSSignatureLength]byte(s.s.Signature)
}

func (s signedExecutionPayloadEnvelopeHeze) IsNil() bool {
	if s.s == nil || len(s.s.Signature) != field_params.BLSSignatureLength || s.s.Message == nil {
		return true
	}
	w := executionPayloadEnvelopeHeze{p: s.s.Message}
	return w.IsNil()
}

func (s signedExecutionPayloadEnvelopeHeze) SigningRoot(domain []byte) ([32]byte, error) {
	return signing.ComputeSigningRoot(s.s.Message, domain)
}

func (s signedExecutionPayloadEnvelopeHeze) Proto() proto.Message {
	return s.s
}

// Version returns the fork the envelope container belongs to.
func (s signedExecutionPayloadEnvelopeHeze) Version() int {
	return version.Heze
}

func (p *executionPayloadEnvelopeHeze) IsNil() bool {
	if p.p == nil || p.p.Payload == nil {
		return true
	}
	return len(p.p.ParentBeaconBlockRoot) != field_params.RootLength
}

func (p *executionPayloadEnvelopeHeze) IsBlinded() bool {
	return false
}

func (p *executionPayloadEnvelopeHeze) Execution() (interfaces.ExecutionData, error) {
	return WrappedExecutionPayloadGloas(p.p.Payload)
}

func (p *executionPayloadEnvelopeHeze) ExecutionRequests() *enginev1.ExecutionRequestsGloas {
	return ethpb.CopyExecutionRequestsGloas(p.p.ExecutionRequests)
}

func (p *executionPayloadEnvelopeHeze) BuilderIndex() primitives.BuilderIndex {
	return p.p.BuilderIndex
}

// BeaconBlockRoot returns the beacon block root the envelope is bound to. The
// binding is attached at wrap time (EIP-8411 removed it from the container).
func (p *executionPayloadEnvelopeHeze) BeaconBlockRoot() [field_params.RootLength]byte {
	return p.root
}

func (p *executionPayloadEnvelopeHeze) ParentBeaconBlockRoot() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.ParentBeaconBlockRoot)
}

func (p *executionPayloadEnvelopeHeze) Slot() primitives.Slot {
	return primitives.Slot(p.p.Payload.SlotNumber)
}

func (p *executionPayloadEnvelopeHeze) BlockHash() [field_params.RootLength]byte {
	return [field_params.RootLength]byte(p.p.Payload.BlockHash)
}

// Version returns the fork the envelope container belongs to.
func (p *executionPayloadEnvelopeHeze) Version() int {
	return version.Heze
}

// RebindEnvelopeRoot returns a wrapper bound to the given beacon block root.
// Gloas envelopes carry their root inline and are returned unchanged; Heze
// wrappers are reconstructed with the new binding.
func RebindEnvelopeRoot(signed interfaces.ROSignedExecutionPayloadEnvelope, root [field_params.RootLength]byte) (interfaces.ROSignedExecutionPayloadEnvelope, error) {
	switch pb := signed.Proto().(type) {
	case *ethpb.SignedExecutionPayloadEnvelope:
		return signed, nil
	case *ethpb.SignedExecutionPayloadEnvelopeHeze:
		return WrappedROSignedExecutionPayloadEnvelopeHeze(pb, root)
	default:
		return nil, errors.New("unknown signed execution payload envelope type")
	}
}
