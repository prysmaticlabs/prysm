//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) waitForChainStart() (*ethpb.ChainStartResponse, error) {
	genesis, err := c.getGenesis()
	if err != nil {
		return nil, err
	}

	genesisTime, err := strconv.ParseUint(genesis.Data.GenesisTime, 10, 64)
	if err != nil {
		return nil, err
	}

	chainStartResponse := &ethpb.ChainStartResponse{}
	chainStartResponse.Started = true
	chainStartResponse.GenesisTime = genesisTime

	// Remove the leading 0x from the string before decoding it to bytes
	genesisValidatorRoot, err := hex.DecodeString(genesis.Data.GenesisValidatorsRoot[2:])
	if err != nil {
		return nil, err
	}
	chainStartResponse.GenesisValidatorsRoot = genesisValidatorRoot

	return chainStartResponse, nil
}

func (c beaconApiValidatorClient) getGenesis() (*rpcmiddleware.GenesisResponseJson, error) {
	resp, err := c.httpClient.Get(c.url + "/eth/v1/beacon/genesis")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errorJson := apimiddleware.DefaultErrorJson{}
		err = json.NewDecoder(resp.Body).Decode(&errorJson)
		if err != nil {
			return nil, err
		}

		return nil, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	genesisJson := &rpcmiddleware.GenesisResponseJson{}
	err = json.NewDecoder(resp.Body).Decode(genesisJson)
	if err != nil {
		return nil, err
	}

	return genesisJson, nil
}
