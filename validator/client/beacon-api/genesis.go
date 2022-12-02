//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) waitForChainStart(ctx context.Context) (*ethpb.ChainStartResponse, error) {
	genesis, httpError, err := c.getGenesis()

	for err != nil {
		if httpError == nil || httpError.Code != http.StatusNotFound {
			return nil, errors.Wrap(err, "failed to get genesis data")
		}

		// Error 404 means that the chain genesis info is not yet known, so we query it every second until it's ready
		select {
		case <-time.After(time.Second):
			genesis, httpError, err = c.getGenesis()
		case <-ctx.Done():
			return nil, errors.New("context canceled")
		}
	}

	genesisTime, err := strconv.ParseUint(genesis.Data.GenesisTime, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse genesis time: %s", genesis.Data.GenesisTime)
	}

	chainStartResponse := &ethpb.ChainStartResponse{}
	chainStartResponse.Started = true
	chainStartResponse.GenesisTime = genesisTime

	if !validRoot(genesis.Data.GenesisValidatorsRoot) {
		return nil, errors.Errorf("invalid genesis validators root: %s", genesis.Data.GenesisValidatorsRoot)
	}

	genesisValidatorRoot, err := hexutil.Decode(genesis.Data.GenesisValidatorsRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode genesis validators root")
	}
	chainStartResponse.GenesisValidatorsRoot = genesisValidatorRoot

	return chainStartResponse, nil
}

func (c beaconApiValidatorClient) getGenesis() (*rpcmiddleware.GenesisResponseJson, *apimiddleware.DefaultErrorJson, error) {
	resp, err := c.httpClient.Get(c.url + "/eth/v1/beacon/genesis")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query REST API genesis endpoint")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errorJson := &apimiddleware.DefaultErrorJson{}
		if err := json.NewDecoder(resp.Body).Decode(&errorJson); err != nil {
			return nil, nil, errors.Wrap(err, "failed to decode response body genesis error json")
		}

		return nil, errorJson, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	genesisJson := &rpcmiddleware.GenesisResponseJson{}
	if err := json.NewDecoder(resp.Body).Decode(&genesisJson); err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode response body genesis json")
	}

	if genesisJson.Data == nil {
		return nil, nil, errors.New("GenesisResponseJson.Data is nil")
	}

	return genesisJson, nil, nil
}
