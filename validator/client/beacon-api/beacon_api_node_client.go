package beacon_api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/config"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/node"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type beaconApiNodeClient struct {
	fallbackClient  iface.NodeClient
	jsonRestHandler JsonRestHandler
	genesisProvider GenesisProvider
}

func (c *beaconApiNodeClient) GetSyncStatus(ctx context.Context, _ *empty.Empty) (*ethpb.SyncStatus, error) {
	syncingResponse := node.SyncStatusResponse{}
	errJson, err := c.jsonRestHandler.Get(ctx, "/eth/v1/node/syncing", &syncingResponse)
	if err != nil {
		return nil, errors.Wrapf(err, msgUnexpectedError)
	}
	if errJson != nil {
		return nil, errJson
	}

	if syncingResponse.Data == nil {
		return nil, errors.New("syncing data is nil")
	}

	return &ethpb.SyncStatus{
		Syncing: syncingResponse.Data.IsSyncing,
	}, nil
}

func (c *beaconApiNodeClient) GetGenesis(ctx context.Context, _ *empty.Empty) (*ethpb.Genesis, error) {
	genesisJson, errJson, err := c.genesisProvider.GetGenesis(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis")
	}
	if errJson != nil {
		return nil, errJson
	}

	genesisValidatorRoot, err := hexutil.Decode(genesisJson.GenesisValidatorsRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode genesis validator root `%s`", genesisJson.GenesisValidatorsRoot)
	}

	genesisTime, err := strconv.ParseInt(genesisJson.GenesisTime, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse genesis time `%s`", genesisJson.GenesisTime)
	}

	depositContractJson := config.GetDepositContractResponse{}
	errJson, err = c.jsonRestHandler.Get(ctx, "/eth/v1/config/deposit_contract", &depositContractJson)
	if err != nil {
		return nil, errors.Wrapf(err, msgUnexpectedError)
	}
	if errJson != nil {
		return nil, errJson
	}

	if depositContractJson.Data == nil {
		return nil, errors.New("deposit contract data is nil")
	}

	depositContactAddress, err := hexutil.Decode(depositContractJson.Data.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode deposit contract address `%s`", depositContractJson.Data.Address)
	}

	return &ethpb.Genesis{
		GenesisTime: &timestamppb.Timestamp{
			Seconds: genesisTime,
		},
		DepositContractAddress: depositContactAddress,
		GenesisValidatorsRoot:  genesisValidatorRoot,
	}, nil
}

func (c *beaconApiNodeClient) GetVersion(ctx context.Context, _ *empty.Empty) (*ethpb.Version, error) {
	var versionResponse node.GetVersionResponse
	errJson, err := c.jsonRestHandler.Get(ctx, "/eth/v1/node/version", &versionResponse)
	if err != nil {
		return nil, errors.Wrapf(err, msgUnexpectedError)
	}
	if errJson != nil {
		return nil, errJson
	}

	if versionResponse.Data == nil || versionResponse.Data.Version == "" {
		return nil, errors.New("empty version response")
	}

	return &ethpb.Version{
		Version: versionResponse.Data.Version,
	}, nil
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
		genesisProvider: beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler},
	}
}
