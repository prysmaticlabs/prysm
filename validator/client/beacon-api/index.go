//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	neturl "net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) validatorIndex(in *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	stringPubKey := hexutil.Encode(in.PublicKey)

	stateValidator, err := c.getStateValidator(stringPubKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator state")
	}

	if len(stateValidator.Data) == 0 {
		return nil, errors.Errorf("could not find validator index for public key `%s`", stringPubKey)
	}

	stringValidatorIndex := stateValidator.Data[0].Index

	index, err := strconv.ParseUint(stringValidatorIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read validator index")
	}

	return &ethpb.ValidatorIndexResponse{Index: types.ValidatorIndex(index)}, nil
}

func (c beaconApiValidatorClient) getStateValidator(pubkey string) (*rpcmiddleware.StateValidatorsResponseJson, error) {
	params := neturl.Values{"id": []string{pubkey}}

	url := buildURL(
		"/eth/v1/beacon/states/head/validators",
		params,
	)

	stateValidatorsJson := &rpcmiddleware.StateValidatorsResponseJson{}
	_, err := c.jsonRestHandler.GetRestJsonResponse(url, stateValidatorsJson)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get json response")
	}

	if stateValidatorsJson.Data == nil {
		return nil, errors.New("stateValidatorsJson.Data is nil")
	}

	return stateValidatorsJson, nil
}
