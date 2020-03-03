package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// SpanDetector defines an interface for Spanners to follow to allow mocks.
type SpanDetector interface {
	// Read functions.
	DetectSlashingForValidator(
		ctx context.Context,
		validatorIdx uint64,
		attData *ethpb.AttestationData,
	) (*types.DetectionResult, error)

	// Write functions.
	UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error
}
