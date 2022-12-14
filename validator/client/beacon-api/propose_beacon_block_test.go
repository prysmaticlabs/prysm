package beacon_api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestProposeBeaconBlock_Error(t *testing.T) {
	testSuites := []struct {
		name                 string
		expectedErrorMessage string
		expectedHttpError    *apimiddleware.DefaultErrorJson
	}{
		{
			name:                 "error 202",
			expectedErrorMessage: "block was successfully broadcasted but failed validation",
			expectedHttpError: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusAccepted,
				Message: "202 error",
			},
		},
		{
			name:                 "request failed",
			expectedErrorMessage: "failed to send POST data to REST endpoint",
			expectedHttpError:    nil,
		},
	}

	testCases := []struct {
		name             string
		consensusVersion string
		endpoint         string
		block            *ethpb.GenericSignedBeaconBlock
	}{
		{
			name:             "phase0",
			consensusVersion: "phase0",
			endpoint:         "/eth/v1/beacon/blocks",
			block: &ethpb.GenericSignedBeaconBlock{
				Block: generateSignedPhase0Block(),
			},
		},
		{
			name:             "altair",
			consensusVersion: "altair",
			endpoint:         "/eth/v1/beacon/blocks",
			block: &ethpb.GenericSignedBeaconBlock{
				Block: generateSignedAltairBlock(),
			},
		},
		{
			name:             "bellatrix",
			consensusVersion: "bellatrix",
			endpoint:         "/eth/v1/beacon/blocks",
			block: &ethpb.GenericSignedBeaconBlock{
				Block: generateSignedBellatrixBlock(),
			},
		},
		{
			name:             "blinded bellatrix",
			consensusVersion: "bellatrix",
			endpoint:         "/eth/v1/beacon/blinded_blocks",
			block: &ethpb.GenericSignedBeaconBlock{
				Block: generateSignedBlindedBellatrixBlock(),
			},
		},
		{
			name:             "blinded capella",
			consensusVersion: "capella",
			endpoint:         "/eth/v1/beacon/blinded_blocks",
			block: &ethpb.GenericSignedBeaconBlock{
				Block: generateSignedBlindedCapellaBlock(),
			},
		},
	}

	for _, testSuite := range testSuites {
		for _, testCase := range testCases {
			t.Run(testSuite.name+"/"+testCase.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

				headers := map[string]string{"Eth-Consensus-Version": testCase.consensusVersion}
				jsonRestHandler.EXPECT().PostRestJson(
					testCase.endpoint,
					headers,
					gomock.Any(),
					nil,
				).Return(
					testSuite.expectedHttpError,
					errors.New("foo error"),
				).Times(1)

				validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
				_, err := validatorClient.proposeBeaconBlock(testCase.block)
				assert.ErrorContains(t, testSuite.expectedErrorMessage, err)
				assert.ErrorContains(t, "foo error", err)
			})
		}
	}
}

func TestProposeBeaconBlock_UnsupportedBlockType(t *testing.T) {
	validatorClient := &beaconApiValidatorClient{}
	_, err := validatorClient.proposeBeaconBlock(&ethpb.GenericSignedBeaconBlock{})
	assert.ErrorContains(t, "unsupported block type", err)
}
