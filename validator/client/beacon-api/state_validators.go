package beacon_api

import (
	"context"
	neturl "net/url"
	"strconv"

	"github.com/pkg/errors"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
)

type stateValidatorsProvider interface {
	GetStateValidators(context.Context, []string, []int64, []string) (*rpcmiddleware.StateValidatorsResponseJson, error)
}

type beaconApiStateValidatorsProvider struct {
	jsonRestHandler jsonRestHandler
}

func (c beaconApiStateValidatorsProvider) GetStateValidators(
	ctx context.Context,
	stringPubkeys []string,
	indexes []int64,
	statuses []string,
) (*rpcmiddleware.StateValidatorsResponseJson, error) {
	params := neturl.Values{}

	stringPubKeysSet := make(map[string]struct{}, len(stringPubkeys))
	indexesSet := make(map[int64]struct{}, len(indexes))

	for _, stringPubkey := range stringPubkeys {
		if _, ok := stringPubKeysSet[stringPubkey]; !ok {
			stringPubKeysSet[stringPubkey] = struct{}{}
			params.Add("id", stringPubkey)
		}
	}

	for _, index := range indexes {
		if _, ok := indexesSet[index]; !ok {
			indexesSet[index] = struct{}{}
			params.Add("id", strconv.FormatInt(index, 10))
		}
	}

	for _, status := range statuses {
		params.Add("status", status)
	}

	url := buildURL(
		"/eth/v1/beacon/states/head/validators",
		params,
	)

	stateValidatorsJson := &rpcmiddleware.StateValidatorsResponseJson{}

	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, url, stateValidatorsJson); err != nil {
		return &rpcmiddleware.StateValidatorsResponseJson{}, errors.Wrap(err, "failed to get json response")
	}

	if stateValidatorsJson.Data == nil {
		return &rpcmiddleware.StateValidatorsResponseJson{}, errors.New("stateValidatorsJson.Data is nil")
	}

	return stateValidatorsJson, nil
}
