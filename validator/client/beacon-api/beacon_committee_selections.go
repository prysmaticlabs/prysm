package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
)

type aggregatedSelectionResponse struct {
	Data []shared.BeaconCommitteeSelection `json:"data"`
}

func (c *beaconApiValidatorClient) getAggregatedSelection(ctx context.Context, selections []shared.BeaconCommitteeSelection) ([]shared.BeaconCommitteeSelection, error) {
	body, err := json.Marshal(selections)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request body selections")
	}

	var resp aggregatedSelectionResponse
	errJson, err := c.jsonRestHandler.Post(ctx, "/eth/v1/validator/beacon_committee_selections", nil, bytes.NewBuffer(body), &resp)
	if err != nil {
		return nil, errors.Wrap(err, "error calling post endpoint")
	}
	if errJson != nil {
		return nil, errJson
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("no aggregated selection returned")
	}

	if len(selections) != len(resp.Data) {
		return nil, errors.New("mismatching number of selections")
	}

	return resp.Data, nil
}
