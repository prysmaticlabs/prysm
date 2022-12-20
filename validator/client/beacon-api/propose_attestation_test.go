package beacon_api

import (
	"bytes"
	"encoding/json"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/test-helpers"
	"testing"
)

func TestProposeAttestation_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	attestation := &ethpb.Attestation{
		AggregationBits: test_helpers.FillByteSlice(4, 74),
		Data: &ethpb.AttestationData{
			Slot:            75,
			CommitteeIndex:  76,
			BeaconBlockRoot: test_helpers.FillByteSlice(32, 38),
			Source: &ethpb.Checkpoint{
				Epoch: 78,
				Root:  test_helpers.FillByteSlice(32, 79),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 80,
				Root:  test_helpers.FillByteSlice(32, 81),
			},
		},
		Signature: test_helpers.FillByteSlice(96, 82),
	}

	marshalledAttestations, err := json.Marshal(jsonifyAttestations([]*ethpb.Attestation{attestation}))
	require.NoError(t, err)

	jsonRestHandler.EXPECT().PostRestJson(
		"/eth/v1/beacon/pool/attestations",
		nil,
		bytes.NewBuffer(marshalledAttestations),
		nil,
	)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	proposeResponse, err := validatorClient.proposeAttestation(attestation)
	require.NoError(t, err)
	require.NotNil(t, proposeResponse)

	expectedAttestationDataRoot, err := attestation.Data.HashTreeRoot()
	require.NoError(t, err)

	// Make sure that the attestation data root is set
	assert.DeepEqual(t, expectedAttestationDataRoot[:], proposeResponse.AttestationDataRoot)
}
