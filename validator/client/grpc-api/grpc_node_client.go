package grpc_api

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	"google.golang.org/grpc"
)

type grpcNodeClient struct {
	beaconNodeValidatorClient ethpb.NodeClient
}

func (c *grpcNodeClient) GetSyncStatus(ctx context.Context, in *empty.Empty) (*ethpb.SyncStatus, error) {
	return c.GetSyncStatus(ctx, in)
}

func (c *grpcNodeClient) GetGenesis(ctx context.Context, in *empty.Empty) (*ethpb.Genesis, error) {
	return c.GetGenesis(ctx, in)
}

func (c *grpcNodeClient) GetVersion(ctx context.Context, in *empty.Empty) (*ethpb.Version, error) {
	return c.GetVersion(ctx, in)
}

func (c *grpcNodeClient) ListImplementedServices(ctx context.Context, in *empty.Empty) (*ethpb.ImplementedServices, error) {
	return c.ListImplementedServices(ctx, in)
}

func (c *grpcNodeClient) GetHost(ctx context.Context, in *empty.Empty) (*ethpb.HostData, error) {
	return c.GetHost(ctx, in)
}

func (c *grpcNodeClient) GetPeer(ctx context.Context, in *ethpb.PeerRequest) (*ethpb.Peer, error) {
	return c.GetPeer(ctx, in)
}

func (c *grpcNodeClient) ListPeers(ctx context.Context, in *empty.Empty) (*ethpb.Peers, error) {
	return c.ListPeers(ctx, in)
}

func (c *grpcNodeClient) GetETH1ConnectionStatus(ctx context.Context, in *empty.Empty) (*ethpb.ETH1ConnectionStatus, error) {
	return c.GetETH1ConnectionStatus(ctx, in)
}

func NewNodeClient(cc grpc.ClientConnInterface) iface.NodeClient {
	return &grpcNodeClient{ethpb.NewNodeClient(cc)}
}
