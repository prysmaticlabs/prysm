package grpc_api

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	"google.golang.org/grpc"
)

type grpcNodeClient struct {
	nodeClient ethpb.NodeClient
}

func (c *grpcNodeClient) GetSyncStatus(ctx context.Context, in *empty.Empty) (*ethpb.SyncStatus, error) {
	return c.nodeClient.GetSyncStatus(ctx, in)
}

func (c *grpcNodeClient) GetGenesis(ctx context.Context, in *empty.Empty) (*ethpb.Genesis, error) {
	return c.nodeClient.GetGenesis(ctx, in)
}

func (c *grpcNodeClient) GetVersion(ctx context.Context, in *empty.Empty) (*ethpb.Version, error) {
	return c.nodeClient.GetVersion(ctx, in)
}

func (c *grpcNodeClient) ListImplementedServices(ctx context.Context, in *empty.Empty) (*ethpb.ImplementedServices, error) {
	return c.nodeClient.ListImplementedServices(ctx, in)
}

func (c *grpcNodeClient) GetHost(ctx context.Context, in *empty.Empty) (*ethpb.HostData, error) {
	return c.nodeClient.GetHost(ctx, in)
}

func (c *grpcNodeClient) GetPeer(ctx context.Context, in *ethpb.PeerRequest) (*ethpb.Peer, error) {
	return c.nodeClient.GetPeer(ctx, in)
}

func (c *grpcNodeClient) ListPeers(ctx context.Context, in *empty.Empty) (*ethpb.Peers, error) {
	return c.nodeClient.ListPeers(ctx, in)
}

func (c *grpcNodeClient) GetETH1ConnectionStatus(ctx context.Context, in *empty.Empty) (*ethpb.ETH1ConnectionStatus, error) {
	return c.nodeClient.GetETH1ConnectionStatus(ctx, in)
}

func NewNodeClient(cc grpc.ClientConnInterface) iface.NodeClient {
	return &grpcNodeClient{ethpb.NewNodeClient(cc)}
}
