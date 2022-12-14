package beacon_api

import (
	neturl "net/url"

	"github.com/pkg/errors"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
)

func (c *beaconApiValidatorClient) getStateValidators(
	stringPubkeys []string,
	statuses []string,
) (*rpcmiddleware.StateValidatorsResponseJson, error) {
	params := neturl.Values{}

	for _, stringPubkey := range stringPubkeys {
		params.Add("id", stringPubkey)
	}

	for _, status := range statuses {
		params.Add("status", status)
	}

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
