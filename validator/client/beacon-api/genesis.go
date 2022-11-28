//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type genesisProvider interface {
	GetGenesis() (*rpcmiddleware.GenesisResponse_GenesisJson, error)
}

type beaconApiGenesisProvider struct {
	httpClient http.Client
	url        string
}

func (c beaconApiValidatorClient) waitForChainStart() (*ethpb.ChainStartResponse, error) {
	genesis, err := c.genesisProvider.GetGenesis()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesis data")
	}

	genesisTime, err := strconv.ParseUint(genesis.GenesisTime, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse genesis time: %s", genesis.GenesisTime)
	}

	chainStartResponse := &ethpb.ChainStartResponse{}
	chainStartResponse.Started = true
	chainStartResponse.GenesisTime = genesisTime

	if !validRoot(genesis.GenesisValidatorsRoot) {
		return nil, errors.Errorf("invalid genesis validators root: %s", genesis.GenesisValidatorsRoot)
	}

	genesisValidatorRoot, err := hexutil.Decode(genesis.GenesisValidatorsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode genesis validators root")
	}
	chainStartResponse.GenesisValidatorsRoot = genesisValidatorRoot

	return chainStartResponse, nil
}

func (c beaconApiGenesisProvider) GetGenesis() (*rpcmiddleware.GenesisResponse_GenesisJson, error) {
	resp, err := c.httpClient.Get(c.url + "/eth/v1/beacon/genesis")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query REST API genesis endpoint")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errorJson := apimiddleware.DefaultErrorJson{}
		if err := json.NewDecoder(resp.Body).Decode(&errorJson); err != nil {
			return nil, errors.Wrap(err, "failed to decode response body genesis error json")
		}

		return nil, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	genesisJson := &rpcmiddleware.GenesisResponseJson{}
	if err := json.NewDecoder(resp.Body).Decode(&genesisJson); err != nil {
		return nil, errors.Wrap(err, "failed to decode response body genesis json")
	}

	if genesisJson.Data == nil {
		return nil, errors.New("genesis data is nil")
	}

	return genesisJson.Data, nil
}
