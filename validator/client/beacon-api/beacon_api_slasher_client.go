package beacon_api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
)

type beaconApiSlasherClient struct {
	fallbackClient  iface.SlasherClient
	jsonRestHandler jsonRestHandler
}

func (c beaconApiSlasherClient) IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation) (*ethpb.AttesterSlashingResponse, error) {
	grpcResult, err := c.fallbackClient.IsSlashableAttestation(ctx, in)
	if err != nil {
		return nil, err
	}

	restResult, err := c.getSlashableAttestations(ctx, in)
	if err != nil {
		return nil, err
	}

	marshalledGrpcResult, err := json.Marshal(grpcResult)
	if err != nil {
		return nil, err
	}

	marshalledRestResult, err := json.Marshal(restResult)
	if err != nil {
		return nil, err
	}

	log.Errorf("***************GRPC RESULT: %s", string(marshalledGrpcResult))
	log.Errorf("***************REST RESULT: %s", string(marshalledRestResult))

	return c.getSlashableAttestations(ctx, in)
}

func (c beaconApiSlasherClient) IsSlashableBlock(ctx context.Context, in *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashingResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.IsSlashableBlock(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiSlasherClient.IsSlashableBlock is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiSlasherClientWithFallback.")
}

// Deprecated: Do not use.
func (c beaconApiSlasherClient) HighestAttestations(ctx context.Context, in *ethpb.HighestAttestationRequest) (*ethpb.HighestAttestationResponse, error) {
	if c.fallbackClient != nil {
		return c.fallbackClient.HighestAttestations(ctx, in)
	}

	// TODO: Implement me
	panic("beaconApiSlasherClient.HighestAttestations is not implemented. To use a fallback client, pass a fallback client as the last argument of NewBeaconApiSlasherClientWithFallback.")
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
