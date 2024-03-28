package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) submitAttestations(ctx context.Context, attestations []*ethpb.Attestation) ([]*ethpb.AttestResponse, error) {
	for _, a := range attestations {
		if err := checkNilAttestation(a); err != nil {
			return nil, err
		}
	}

	marshalledAttestation, err := json.Marshal(jsonifyAttestations(attestations))
	if err != nil {
		return nil, err
	}

	if err = c.jsonRestHandler.Post(
		ctx,
		"/eth/v1/beacon/pool/attestations",
		nil,
		bytes.NewBuffer(marshalledAttestation),
		nil,
	); err != nil {
		return nil, err
	}

	resp := make([]*ethpb.AttestResponse, len(attestations))
	for i, a := range attestations {
		attestationDataRoot, err := a.Data.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "failed to compute attestation data root")
		}
		resp[i] = &ethpb.AttestResponse{AttestationDataRoot: attestationDataRoot[:]}
	}
	return resp, nil
}

// checkNilAttestation returns error if attestation or any field of attestation is nil.
func checkNilAttestation(attestation *ethpb.Attestation) error {
	if attestation == nil {
		return errors.New("attestation is nil")
	}

	if attestation.Data == nil {
		return errors.New("attestation data is nil")
	}

	if attestation.Data.Source == nil || attestation.Data.Target == nil {
		return errors.New("source/target in attestation data is nil")
	}

	if len(attestation.AggregationBits) == 0 {
		return errors.New("attestation aggregation bits is empty")
	}

	if len(attestation.Signature) == 0 {
		return errors.New("attestation signature is empty")
	}

	return nil
}
