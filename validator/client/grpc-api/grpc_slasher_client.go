package grpc_api

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	"google.golang.org/grpc"
)

type grpcSlasherClient struct {
	slasherClient ethpb.SlasherClient
}

func (c *grpcSlasherClient) IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation) (*ethpb.AttesterSlashingResponse, error) {
	return c.slasherClient.IsSlashableAttestation(ctx, in)
}

func (c *grpcSlasherClient) IsSlashableBlock(ctx context.Context, in *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashingResponse, error) {
	return c.slasherClient.IsSlashableBlock(ctx, in)
}

func (c *grpcSlasherClient) HighestAttestations(ctx context.Context, in *ethpb.HighestAttestationRequest) (*ethpb.HighestAttestationResponse, error) {
	return c.slasherClient.HighestAttestations(ctx, in)
}

func NewSlasherClient(cc grpc.ClientConnInterface) iface.SlasherClient {
	return &grpcSlasherClient{ethpb.NewSlasherClient(cc)}
}
