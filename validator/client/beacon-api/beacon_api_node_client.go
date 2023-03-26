package beacon_api

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
)

type beaconApiNodeClient struct {
	fallbackClient  iface.NodeClient
	jsonRestHandler jsonRestHandler
}

func (c *beaconApiNodeClient) GetSyncStatus(ctx context.Context, _ *empty.Empty) (*ethpb.SyncStatus, error) {
	syncingResponse := apimiddleware.SyncingResponseJson{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, "/eth/v1/node/syncing", &syncingResponse); err != nil {
		return nil, errors.Wrap(err, "failed to get sync status")
	}

	if syncingResponse.Data == nil {
		return nil, errors.New("syncing data is nil")
	}

	return &ethpb.SyncStatus{
		Syncing: syncingResponse.Data.IsSyncing,
	}, nil
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

func (c *beaconApiNodeClient) ListPeers(ctx context.Context, in *empty.Empty) (*ethpb.Peers, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.ListPeers(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiNodeClient.ListPeers is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiNodeClientWithFallback.")
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
