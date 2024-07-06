package payloadattestation

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	log "github.com/sirupsen/logrus"
)

// Verifier defines the methods required for verifying payload attestation messages.
type Verifier interface {
	VerifyCurrentSlot() error
	VerifyPayloadMessageAlreadySeen(error)
	VerifyKnownPayloadStatus() error
	VerifyValidatorInPayload() error
	VerifyBlockRootSeen() error
	VerifyBlockRootValid() error
	VerifySignatureValid() error
	MeetRequirement(Requirement)
	GetVerifiedPayloadAttestationMessage() (VerifiedReadOnly, error)
}

// Resources contains the resources shared across verifiers.
type Resources struct {
	clock *startup.Clock
}

// NewVerifierFunc is a factory function to create a new Verifier.
type NewVerifierFunc func(r ReadOnlyPayloadAtt, reqs []Requirement) Verifier

// PayloadVerifier is a read-only verifier for payload attestation messages.
type PayloadVerifier struct {
	Resources
	payloadAtt ReadOnlyPayloadAtt
}

// VerifyCurrentSlot verifies if the current slot matches the expected slot.
func (v *PayloadVerifier) VerifyCurrentSlot() error {
	if v.clock.CurrentSlot() != v.clock.CurrentSlot() {
		log.WithFields(fields(v.payloadAtt)).Errorf("does not match current slot %d", v.clock.CurrentSlot())
		return ErrMismatchCurrentSlot
	}
	return nil
}

// VerifyKnownPayloadStatus verifies if the payload status is known.
func (v *PayloadVerifier) VerifyKnownPayloadStatus() error {
	if v.payloadAtt.PayloadStatus() > primitives.PAYLOAD_INVALID_STATUS {
		log.WithFields(fields(v.payloadAtt)).Error("unknown payload status")
		return ErrUnknownPayloadStatus
	}
	return nil
}
