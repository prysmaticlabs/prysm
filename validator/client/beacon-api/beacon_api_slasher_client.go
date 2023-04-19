package beacon_api

import (
	"context"
	"net/http"
	"time"

	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
)

type beaconApiSlasherClient struct {
	fallbackClient  iface.SlasherClient
	jsonRestHandler jsonRestHandler
}

func (c beaconApiSlasherClient) IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation) (*ethpb.AttesterSlashingResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.IsSlashableAttestation(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiSlasherClient.IsSlashableAttestation is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiSlasherClientWithFallback.")
}

func (c beaconApiSlasherClient) IsSlashableBlock(ctx context.Context, in *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashingResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.IsSlashableBlock(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiSlasherClient.IsSlashableBlock is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiSlasherClientWithFallback.")
}

func NewSlasherClientWithFallback(host string, timeout time.Duration, fallbackClient iface.SlasherClient) iface.SlasherClient {
	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: timeout},
		host:       host,
	}

	return &beaconApiSlasherClient{
		jsonRestHandler: jsonRestHandler,
		fallbackClient:  fallbackClient,
	}
}
