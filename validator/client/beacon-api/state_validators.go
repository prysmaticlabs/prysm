package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

type StateValidatorsProvider interface {
	GetStateValidators(context.Context, []string, []primitives.ValidatorIndex, []string) (*structs.GetValidatorsResponse, error)
	GetStateValidatorsForSlot(context.Context, primitives.Slot, []string, []primitives.ValidatorIndex, []string) (*structs.GetValidatorsResponse, error)
	GetStateValidatorsForHead(context.Context, []string, []primitives.ValidatorIndex, []string) (*structs.GetValidatorsResponse, error)
}

type beaconApiStateValidatorsProvider struct {
	jsonRestHandler JsonRestHandler
}

func (c beaconApiStateValidatorsProvider) GetStateValidators(
	ctx context.Context,
	stringPubkeys []string,
	indexes []primitives.ValidatorIndex,
	statuses []string,
) (*structs.GetValidatorsResponse, error) {
	stringIndices := convertValidatorIndicesToStrings(indexes)
	return c.getStateValidatorsHelper(ctx, "/eth/v1/beacon/states/head/validators", append(stringIndices, stringPubkeys...), statuses)
}

func (c beaconApiStateValidatorsProvider) GetStateValidatorsForSlot(
	ctx context.Context,
	slot primitives.Slot,
	stringPubkeys []string,
	indices []primitives.ValidatorIndex,
	statuses []string,
) (*structs.GetValidatorsResponse, error) {
	stringIndices := convertValidatorIndicesToStrings(indices)
	url := fmt.Sprintf("/eth/v1/beacon/states/%d/validators", slot)
	return c.getStateValidatorsHelper(ctx, url, append(stringIndices, stringPubkeys...), statuses)
}

func (c beaconApiStateValidatorsProvider) GetStateValidatorsForHead(
	ctx context.Context,
	stringPubkeys []string,
	indices []primitives.ValidatorIndex,
	statuses []string,
) (*structs.GetValidatorsResponse, error) {
	stringIndices := convertValidatorIndicesToStrings(indices)
	return c.getStateValidatorsHelper(ctx, "/eth/v1/beacon/states/head/validators", append(stringIndices, stringPubkeys...), statuses)
}

func convertValidatorIndicesToStrings(indices []primitives.ValidatorIndex) []string {
	var result []string
	indicesSet := make(map[primitives.ValidatorIndex]struct{}, len(indices))
	for _, index := range indices {
		if _, ok := indicesSet[index]; !ok {
			indicesSet[index] = struct{}{}
			result = append(result, strconv.FormatUint(uint64(index), 10))
		}
	}
	return result
}

func (c beaconApiStateValidatorsProvider) getStateValidatorsHelper(
	ctx context.Context,
	endpoint string,
	vals []string,
	statuses []string,
) (*structs.GetValidatorsResponse, error) {
	req := structs.GetValidatorsRequest{
		Ids:      []string{},
		Statuses: []string{},
	}
	req.Statuses = append(req.Statuses, statuses...)

	valSet := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		if _, ok := valSet[v]; !ok {
			valSet[v] = struct{}{}
			req.Ids = append(req.Ids, v)
		}
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal request into JSON")
	}
	stateValidatorsJson := &structs.GetValidatorsResponse{}
	// First try POST endpoint to check whether it is supported by the beacon node.
	if err = c.jsonRestHandler.Post(ctx, endpoint, nil, bytes.NewBuffer(reqBytes), stateValidatorsJson); err == nil {
		if stateValidatorsJson.Data == nil {
			return nil, errors.New("stateValidatorsJson.Data is nil")
		}

		return stateValidatorsJson, nil
	}

	// Re-initialise the response just in case.
	stateValidatorsJson = &structs.GetValidatorsResponse{}

	// Seems like POST isn't supported by the beacon node, let's try the GET one.
	queryParams := url.Values{}
	for _, id := range req.Ids {
		queryParams.Add("id", id)
	}
	for _, st := range req.Statuses {
		queryParams.Add("status", st)
	}

	query := buildURL(endpoint, queryParams)

	err = c.jsonRestHandler.Get(ctx, query, stateValidatorsJson)
	if err != nil {
		return nil, err
	}

	if stateValidatorsJson.Data == nil {
		return nil, errors.New("stateValidatorsJson.Data is nil")
	}

	return stateValidatorsJson, nil
}
