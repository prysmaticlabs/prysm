package beacon_api

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
)

type beaconApiNodeClient struct {
	fallbackClient  iface.NodeClient
	jsonRestHandler jsonRestHandler
}

func (c *beaconApiNodeClient) GetSyncStatus(ctx context.Context, in *empty.Empty) (*ethpb.SyncStatus, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetSyncStatus(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.GetSyncStatus is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) GetGenesis(ctx context.Context, in *empty.Empty) (*ethpb.Genesis, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetGenesis(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.GetGenesis is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) GetVersion(ctx context.Context, in *empty.Empty) (*ethpb.Version, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetVersion(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.GetVersion is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) ListImplementedServices(ctx context.Context, in *empty.Empty) (*ethpb.ImplementedServices, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListImplementedServices(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.ListImplementedServices is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) GetHost(ctx context.Context, in *empty.Empty) (*ethpb.HostData, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetHost(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.GetHost is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) GetPeer(ctx context.Context, in *ethpb.PeerRequest) (*ethpb.Peer, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetPeer(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.GetPeer is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) ListPeers(ctx context.Context, in *empty.Empty) (*ethpb.Peers, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListPeers(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.ListPeers is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func (c *beaconApiNodeClient) GetETH1ConnectionStatus(ctx context.Context, in *empty.Empty) (*ethpb.ETH1ConnectionStatus, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.GetETH1ConnectionStatus(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.GetETH1ConnectionStatus is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
}

func NewNodeClientWithFallback(host string, timeout time.Duration, fallbackClient iface.NodeClient) iface.NodeClient {
	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: timeout},
		host:       host,
	}

	return &beaconApiNodeClient{
		jsonRestHandler: jsonRestHandler,
		fallbackClient:  fallbackClient,
	}
}
