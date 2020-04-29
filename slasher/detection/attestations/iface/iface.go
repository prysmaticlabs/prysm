// Package iface defines an interface for a slashable attestation detector struct.
package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// SpanDetector defines an interface for Spanners to follow to allow mocks.
type SpanDetector interface {
	// Read functions.
	DetectSlashingsForAttestation(
		ctx context.Context,
		att *ethpb.IndexedAttestation,
	) ([]*types.DetectionResult, error)

	// Write functions.
	UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error
}
