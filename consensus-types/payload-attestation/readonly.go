package payloadattestation

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// ReadOnlyPayloadAtt represents a read-only payload attestation message.
type ReadOnlyPayloadAtt struct {
	message *ethpb.PayloadAttestationMessage
}

// validatePayload checks if the given payload attestation message is valid.
func validatePayload(message *ethpb.PayloadAttestationMessage) error {
	if message == nil {
		return errNilPayloadMessage
	}
	if message.Data == nil {
		return errNilPayloadData
	}
	if len(message.Signature) == 0 {
		return errMissingPayloadSignature
	}
	return nil
}

// NewReadOnly creates a new ReadOnlyPayloadAtt instance after validating the message.
func NewReadOnly(message *ethpb.PayloadAttestationMessage) (ReadOnlyPayloadAtt, error) {
	if err := validatePayload(message); err != nil {
		return ReadOnlyPayloadAtt{}, err
	}
	return ReadOnlyPayloadAtt{message}, nil
}

// ValidatorIndex returns the validator index from the payload attestation message.
func (r *ReadOnlyPayloadAtt) ValidatorIndex() primitives.ValidatorIndex {
	return r.message.ValidatorIndex
}

// Signature returns the signature from the payload attestation message.
func (r *ReadOnlyPayloadAtt) Signature() [96]byte {
	return bytesutil.ToBytes96(r.message.Signature)
}

// BeaconBlockRoot returns the beacon block root from the payload attestation message.
func (r *ReadOnlyPayloadAtt) BeaconBlockRoot() [32]byte {
	return bytesutil.ToBytes32(r.message.Data.BeaconBlockRoot)
}

// Slot returns the slot from the payload attestation message.
func (r *ReadOnlyPayloadAtt) Slot() primitives.Slot {
	return r.message.Data.Slot
}

// PayloadStatus returns the payload status from the payload attestation message.
func (r *ReadOnlyPayloadAtt) PayloadStatus() primitives.PTCStatus {
	return r.message.Data.PayloadStatus
}

// VerifiedReadOnly represents a verified read-only payload attestation message.
type VerifiedReadOnly struct {
	ReadOnlyPayloadAtt
}
