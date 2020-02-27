package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

type SpanDetector interface {
	// Read functions.
	DetectSlashingForValidator(
		ctx context.Context,
		validatorIdx uint64,
		attData *ethpb.AttestationData,
	) (*types.DetectionResult, error)
	SpanForEpochByValidator(ctx context.Context, valIdx uint64, epoch uint64) ([2]uint16, error)
	ValidatorSpansByEpoch(ctx context.Context, epoch uint64) map[uint64][2]uint16

	// Write functions.
	UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error
	DeleteValidatorSpansByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error
}
