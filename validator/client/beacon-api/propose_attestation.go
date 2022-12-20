package beacon_api

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) proposeAttestation(in *ethpb.Attestation) (*ethpb.AttestResponse, error) {
	marshalledAttestation, err := json.Marshal(jsonifyAttestations([]*ethpb.Attestation{in}))
	if err != nil {
		return nil, err
	}

	if _, err := c.jsonRestHandler.PostRestJson("/eth/v1/beacon/pool/attestations", nil, bytes.NewBuffer(marshalledAttestation), nil); err != nil {
		return nil, errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	attestationDataRoot, err := in.Data.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute attestation data root")
	}

	return &ethpb.AttestResponse{AttestationDataRoot: attestationDataRoot[:]}, nil
}
