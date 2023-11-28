package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) submitSignedAggregateSelectionProof(ctx context.Context, in *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	body, err := json.Marshal([]*apimiddleware.SignedAggregateAttestationAndProofJson{jsonifySignedAggregateAndProof(in.SignedAggregateAndProof)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal SignedAggregateAttestationAndProof")
	}

	errJson, err := c.jsonRestHandler.Post(ctx, "/eth/v1/validator/aggregate_and_proofs", nil, bytes.NewBuffer(body), nil)
	if err != nil {
		return nil, errors.Wrapf(err, msgUnexpectedError)
	}
	if errJson != nil {
		return nil, errJson
	}

	attestationDataRoot, err := in.SignedAggregateAndProof.Message.Aggregate.Data.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute attestation data root")
	}

	return &ethpb.SignedAggregateSubmitResponse{AttestationDataRoot: attestationDataRoot[:]}, nil
}
