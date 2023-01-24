package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) submitValidatorRegistrations(ctx context.Context, registrations []*ethpb.SignedValidatorRegistrationV1) error {
	const endpoint = "/eth/v1/validator/register_validator"

	jsonRegistration := make([]*apimiddleware.SignedValidatorRegistrationJson, len(registrations))

	for index, registration := range registrations {
		inMessage := registration.Message
		outMessage := &apimiddleware.ValidatorRegistrationJson{
			FeeRecipient: hexutil.Encode(inMessage.FeeRecipient),
			GasLimit:     strconv.FormatUint(inMessage.GasLimit, 10),
			Timestamp:    strconv.FormatUint(inMessage.Timestamp, 10),
			Pubkey:       hexutil.Encode(inMessage.Pubkey),
		}

		jsonRegistration[index] = &apimiddleware.SignedValidatorRegistrationJson{
			Message:   outMessage,
			Signature: hexutil.Encode(registration.Signature),
		}
	}

	marshalledJsonRegistration, err := json.Marshal(jsonRegistration)
	if err != nil {
		return errors.Wrap(err, "failed to marshal registration")
	}

	if _, err := c.jsonRestHandler.PostRestJson(ctx, endpoint, nil, bytes.NewBuffer(marshalledJsonRegistration), nil); err != nil {
		return errors.Wrapf(err, "failed to send POST data to `%s` REST endpoint", endpoint)
	}

	return nil
}
