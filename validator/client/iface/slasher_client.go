package iface

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type SlasherClient interface {
	IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation) (*ethpb.AttesterSlashingResponse, error)
	IsSlashableBlock(ctx context.Context, in *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashingResponse, error)
	HighestAttestations(ctx context.Context, in *ethpb.HighestAttestationRequest) (*ethpb.HighestAttestationResponse, error)
}
