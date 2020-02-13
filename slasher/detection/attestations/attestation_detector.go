package attestations

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/slasher/detection"
)

// AttDetector common for all slashing detectors.
type AttDetector struct {
	slashingDetector *detection.SlashingDetector
}

// AttSlashingDetector an interface with basic functions that are needed to
// create an attestation slashing detector.
type AttSlashingDetector interface {
	// DoubleVotes looks up db for slashable attesting data that were preformed by the same validator.
	DoubleVotes(
		validatorIdx uint64,
		dataRoot []byte,
		origAtt *ethpb.IndexedAttestation,
	) ([]*ethpb.AttesterSlashing, error)
	// DetectSurroundVotes is a method used to return the attestation that were detected
	// by min max surround detection method.
	DetectSurroundVotes(ctx context.Context, validatorIdx uint64, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
	// DetectAttestationForSlashings returns an attester slashing if the attestation submitted
	// is a slashable vote.
	DetectAttestationForSlashings(ctx context.Context, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
}
