package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) submitValidatorRegistrations(ctx context.Context, registrations []*ethpb.SignedValidatorRegistrationV1) error {
	const endpoint = "/eth/v1/validator/register_validator"

	jsonRegistration := make([]*shared.SignedValidatorRegistration, len(registrations))

	for index, registration := range registrations {
		outMessage, err := shared.SignedValidatorRegistrationFromConsensus(registration)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to encode to SignedValidatorRegistration at index %d", index))
		}
		jsonRegistration[index] = outMessage
	}

	marshalledJsonRegistration, err := json.Marshal(jsonRegistration)
	if err != nil {
		return errors.Wrap(err, "failed to marshal registration")
	}

	errJson, err := c.jsonRestHandler.Post(ctx, endpoint, nil, bytes.NewBuffer(marshalledJsonRegistration), nil)
	if err != nil {
		return errors.Wrapf(err, msgUnexpected)
	}
	if errJson != nil {
		return errors.Wrap(errJson, msgRequestFailed)
	}

	return nil
}
