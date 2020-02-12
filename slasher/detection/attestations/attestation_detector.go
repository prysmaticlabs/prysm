package attestations

import (
	"context"

	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/slasher/detection"
)

type AttDetector struct {
	slashingDetector *detection.SlashingDetector
}

type AttSlashingDetector interface {
	DoubleVotes(
		validatorIdx uint64,
		dataRoot []byte,
		origAtt *ethpb.IndexedAttestation,
	) ([]*ethpb.AttesterSlashing, error)
	DetectSurroundVotes(ctx context.Context, validatorIdx uint64, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
	IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error)
}
