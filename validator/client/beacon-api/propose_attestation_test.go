package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v5/validator/client/beacon-api/test-helpers"
	"go.uber.org/mock/gomock"
)

func TestProposeAttestations(t *testing.T) {
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

	tests := []struct {
		name                 string
		attestation          *ethpb.Attestation
		expectedErrorMessage string
		endpointError        error
		endpointCall         int
	}{
		{
			name:         "valid",
			attestation:  attestation,
			endpointCall: 1,
		},
		{
			name:                 "nil attestation",
			expectedErrorMessage: "attestation is nil",
		},
		{
			name: "nil attestation data",
			attestation: &ethpb.Attestation{
				AggregationBits: test_helpers.FillByteSlice(4, 74),
				Signature:       test_helpers.FillByteSlice(96, 82),
			},
			expectedErrorMessage: "attestation data is nil",
		},
		{
			name: "nil source checkpoint",
			attestation: &ethpb.Attestation{
				AggregationBits: test_helpers.FillByteSlice(4, 74),
				Data: &ethpb.AttestationData{
					Target: &ethpb.Checkpoint{},
				},
				Signature: test_helpers.FillByteSlice(96, 82),
			},
			expectedErrorMessage: "source/target in attestation data is nil",
		},
		{
			name: "nil target checkpoint",
			attestation: &ethpb.Attestation{
				AggregationBits: test_helpers.FillByteSlice(4, 74),
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{},
				},
				Signature: test_helpers.FillByteSlice(96, 82),
			},
			expectedErrorMessage: "source/target in attestation data is nil",
		},
		{
			name: "nil aggregation bits",
			attestation: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{},
					Target: &ethpb.Checkpoint{},
				},
				Signature: test_helpers.FillByteSlice(96, 82),
			},
			expectedErrorMessage: "attestation aggregation bits is empty",
		},
		{
			name: "nil signature",
			attestation: &ethpb.Attestation{
				AggregationBits: test_helpers.FillByteSlice(4, 74),
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{},
					Target: &ethpb.Checkpoint{},
				},
			},
			expectedErrorMessage: "attestation signature is empty",
		},
		{
			name:                 "bad request",
			attestation:          attestation,
			expectedErrorMessage: "bad request",
			endpointError:        errors.New("bad request"),
			endpointCall:         1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			jsonRestHandler := mock.NewMockJsonRestHandler(ctrl)

			var marshalledAttestations []byte
			if checkNilAttestation(test.attestation) == nil {
				b, err := json.Marshal(jsonifyAttestations([]*ethpb.Attestation{test.attestation}))
				require.NoError(t, err)
				marshalledAttestations = b
			}

			ctx := context.Background()

			jsonRestHandler.EXPECT().Post(
				ctx,
				"/eth/v1/beacon/pool/attestations",
				nil,
				bytes.NewBuffer(marshalledAttestations),
				nil,
			).Return(
				test.endpointError,
			).Times(test.endpointCall)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			proposeResponse, err := validatorClient.submitAttestations(ctx, []*ethpb.Attestation{test.attestation})
			if test.expectedErrorMessage != "" {
				require.ErrorContains(t, test.expectedErrorMessage, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, proposeResponse)

			expectedAttestationDataRoot, err := attestation.Data.HashTreeRoot()
			require.NoError(t, err)

			// Make sure that the attestation data root is set
			assert.DeepEqual(t, expectedAttestationDataRoot[:], proposeResponse[0].AttestationDataRoot)
		})
	}
}

func TestProposeAttestations_Multiple(t *testing.T) {
	att1 := &ethpb.Attestation{
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
	att2 := &ethpb.Attestation{
		AggregationBits: test_helpers.FillByteSlice(4, 174),
		Data: &ethpb.AttestationData{
			Slot:            175,
			CommitteeIndex:  176,
			BeaconBlockRoot: test_helpers.FillByteSlice(32, 138),
			Source: &ethpb.Checkpoint{
				Epoch: 178,
				Root:  test_helpers.FillByteSlice(32, 179),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 180,
				Root:  test_helpers.FillByteSlice(32, 181),
			},
		},
		Signature: test_helpers.FillByteSlice(96, 182),
	}

	ctrl := gomock.NewController(t)
	jsonRestHandler := mock.NewMockJsonRestHandler(ctrl)

	marshalledAttestations, err := json.Marshal(jsonifyAttestations([]*ethpb.Attestation{att1, att2}))
	require.NoError(t, err)

	ctx := context.Background()

	jsonRestHandler.EXPECT().Post(
		ctx,
		"/eth/v1/beacon/pool/attestations",
		nil,
		bytes.NewBuffer(marshalledAttestations),
		nil,
	).Return(nil)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	proposeResponse, err := validatorClient.submitAttestations(ctx, []*ethpb.Attestation{att1, att2})
	require.NoError(t, err)
	require.NotNil(t, proposeResponse)
	expectedRoot1, err := att1.Data.HashTreeRoot()
	require.NoError(t, err)
	expectedRoot2, err := att2.Data.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, 2, len(proposeResponse))
	assert.DeepEqual(t, expectedRoot1[:], proposeResponse[0].AttestationDataRoot)
	assert.DeepEqual(t, expectedRoot2[:], proposeResponse[1].AttestationDataRoot)
}
