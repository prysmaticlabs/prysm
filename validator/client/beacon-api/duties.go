package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

type dutiesProvider interface {
	GetAttesterDuties(ctx context.Context, epoch types.Epoch, validatorIndices []types.ValidatorIndex) ([]*apimiddleware.AttesterDutyJson, error)
}

type beaconApiDutiesProvider struct {
	jsonRestHandler jsonRestHandler
}

func (c beaconApiDutiesProvider) GetAttesterDuties(ctx context.Context, epoch types.Epoch, validatorIndices []types.ValidatorIndex) ([]*apimiddleware.AttesterDutyJson, error) {

	jsonValidatorIndices := make([]string, len(validatorIndices))
	for index, validatorIndex := range validatorIndices {
		jsonValidatorIndices[index] = strconv.FormatUint(uint64(validatorIndex), 10)
	}

	validatorIndicesBytes, err := json.Marshal(jsonValidatorIndices)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal validator indices")
	}

	attesterDuties := &apimiddleware.AttesterDutiesResponseJson{}
	if _, err := c.jsonRestHandler.PostRestJson(ctx, fmt.Sprintf("/eth/v1/validator/duties/attester/%d", epoch), nil, bytes.NewBuffer(validatorIndicesBytes), attesterDuties); err != nil {
		return nil, errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	for index, attesterDuty := range attesterDuties.Data {
		if attesterDuty == nil {
			return nil, errors.Errorf("attester duty at index `%d` is nil", index)
		}
	}

	return attesterDuties.Data, nil
}
