package grpc_api

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	"google.golang.org/grpc"
)

type grpcSlasherClient struct {
	beaconNodeValidatorClient ethpb.SlasherClient
}

func (c *grpcSlasherClient) IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation) (*ethpb.AttesterSlashingResponse, error) {
	return c.IsSlashableAttestation(ctx, in)
}

func (c *grpcSlasherClient) IsSlashableBlock(ctx context.Context, in *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashingResponse, error) {
	return c.IsSlashableBlock(ctx, in)
}

func (c *grpcSlasherClient) HighestAttestations(ctx context.Context, in *ethpb.HighestAttestationRequest) (*ethpb.HighestAttestationResponse, error) {
	return c.HighestAttestations(ctx, in)
}

func NewSlasherClient(cc grpc.ClientConnInterface) iface.SlasherClient {
	return &grpcSlasherClient{ethpb.NewSlasherClient(cc)}
}
