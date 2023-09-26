package beacon_api

import (
	"context"
	"fmt"
	neturl "net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type stateValidatorsProvider interface {
	GetStateValidators(context.Context, []string, []int64, []string) (*beacon.GetValidatorsResponse, error)
	GetStateValidatorsForSlot(context.Context, primitives.Slot, []string, []primitives.ValidatorIndex, []string) (*beacon.GetValidatorsResponse, error)
	GetStateValidatorsForHead(context.Context, []string, []primitives.ValidatorIndex, []string) (*beacon.GetValidatorsResponse, error)
}

type beaconApiStateValidatorsProvider struct {
	jsonRestHandler jsonRestHandler
}

func (c beaconApiStateValidatorsProvider) GetStateValidators(
	ctx context.Context,
	stringPubkeys []string,
	indexes []int64,
	statuses []string,
) (*beacon.GetValidatorsResponse, error) {
	params := neturl.Values{}
	indexesSet := make(map[int64]struct{}, len(indexes))
	for _, index := range indexes {
		if _, ok := indexesSet[index]; !ok {
			indexesSet[index] = struct{}{}
			params.Add("id", strconv.FormatInt(index, 10))
		}
	}

	return c.getStateValidatorsHelper(ctx, "/eth/v1/beacon/states/head/validators", params, stringPubkeys, statuses)
}

func (c beaconApiStateValidatorsProvider) GetStateValidatorsForSlot(
	ctx context.Context,
	slot primitives.Slot,
	stringPubkeys []string,
	indices []primitives.ValidatorIndex,
	statuses []string,
) (*beacon.GetValidatorsResponse, error) {
	params := convertValidatorIndicesToParams(indices)
	url := fmt.Sprintf("/eth/v1/beacon/states/%d/validators", slot)
	return c.getStateValidatorsHelper(ctx, url, params, stringPubkeys, statuses)
}

func (c beaconApiStateValidatorsProvider) GetStateValidatorsForHead(
	ctx context.Context,
	stringPubkeys []string,
	indices []primitives.ValidatorIndex,
	statuses []string,
) (*beacon.GetValidatorsResponse, error) {
	params := convertValidatorIndicesToParams(indices)
	return c.getStateValidatorsHelper(ctx, "/eth/v1/beacon/states/head/validators", params, stringPubkeys, statuses)
}

func convertValidatorIndicesToParams(indices []primitives.ValidatorIndex) neturl.Values {
	params := neturl.Values{}
	indicesSet := make(map[primitives.ValidatorIndex]struct{}, len(indices))
	for _, index := range indices {
		if _, ok := indicesSet[index]; !ok {
			indicesSet[index] = struct{}{}
			params.Add("id", strconv.FormatUint(uint64(index), 10))
		}
	}
	return params
}

func (c beaconApiStateValidatorsProvider) getStateValidatorsHelper(
	ctx context.Context,
	endpoint string,
	params neturl.Values,
	stringPubkeys []string,
	statuses []string,
) (*beacon.GetValidatorsResponse, error) {
	stringPubKeysSet := make(map[string]struct{}, len(stringPubkeys))

	for _, stringPubkey := range stringPubkeys {
		if _, ok := stringPubKeysSet[stringPubkey]; !ok {
			stringPubKeysSet[stringPubkey] = struct{}{}
			params.Add("id", stringPubkey)
		}
	}

	for _, status := range statuses {
		params.Add("status", status)
	}

	url := buildURL(endpoint, params)
	stateValidatorsJson := &beacon.GetValidatorsResponse{}

	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, url, stateValidatorsJson); err != nil {
		return &beacon.GetValidatorsResponse{}, errors.Wrap(err, "failed to get json response")
	}

	if stateValidatorsJson.Data == nil {
		return &beacon.GetValidatorsResponse{}, errors.New("stateValidatorsJson.Data is nil")
	}

	return stateValidatorsJson, nil
}
