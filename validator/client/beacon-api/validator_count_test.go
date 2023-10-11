package beacon_api

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	validator2 "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/validator"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
)

func TestGetValidatorCount(t *testing.T) {
	const nodeVersion = "prysm/v0.0.1"

	testCases := []struct {
		name                        string
		versionEndpointError        error
		validatorCountEndpointError error
		versionResponse             apimiddleware.VersionResponseJson
		validatorCountResponse      validator2.ValidatorCountResponse
		validatorCountCalled        int
		expectedResponse            []iface.ValidatorCount
		expectedError               string
	}{
		{
			name: "success",
			versionResponse: apimiddleware.VersionResponseJson{
				Data: &apimiddleware.VersionJson{Version: nodeVersion},
			},
			validatorCountResponse: validator2.ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: []*validator2.ValidatorCount{
					{
						Status: "active",
						Count:  "10",
					},
				},
			},
			validatorCountCalled: 1,
			expectedResponse: []iface.ValidatorCount{
				{
					Status: "active",
					Count:  10,
				},
			},
		},
		{
			name: "not supported beacon node",
			versionResponse: apimiddleware.VersionResponseJson{
				Data: &apimiddleware.VersionJson{Version: "lighthouse/v0.0.1"},
			},
			expectedError: "endpoint not supported",
		},
		{
			name:                 "fails to get version",
			versionEndpointError: errors.New("foo error"),
			expectedError:        "failed to get node version",
		},
		{
			name: "fails to get validator count",
			versionResponse: apimiddleware.VersionResponseJson{
				Data: &apimiddleware.VersionJson{Version: nodeVersion},
			},
			validatorCountEndpointError: errors.New("foo error"),
			validatorCountCalled:        1,
			expectedError:               "foo error",
		},
		{
			name: "nil validator count data",
			versionResponse: apimiddleware.VersionResponseJson{
				Data: &apimiddleware.VersionJson{Version: nodeVersion},
			},
			validatorCountResponse: validator2.ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data:                nil,
			},
			validatorCountCalled: 1,
			expectedError:        "validator count data is nil",
		},
		{
			name: "invalid validator count",
			versionResponse: apimiddleware.VersionResponseJson{
				Data: &apimiddleware.VersionJson{Version: nodeVersion},
			},
			validatorCountResponse: validator2.ValidatorCountResponse{
				ExecutionOptimistic: "false",
				Finalized:           "true",
				Data: []*validator2.ValidatorCount{
					{
						Status: "active",
						Count:  "10",
					},
					{
						Status: "exited",
						Count:  "10",
					},
				},
			},
			validatorCountCalled: 1,
			expectedError:        "mismatch between validator count data and the number of statuses provided",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

			// Expect node version endpoint call.
			var nodeVersionResponse apimiddleware.VersionResponseJson
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				"/eth/v1/node/version",
				&nodeVersionResponse,
			).Return(
				nil,
				test.versionEndpointError,
			).SetArg(
				2,
				test.versionResponse,
			)

			var validatorCountResponse validator2.ValidatorCountResponse
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				"/eth/v1/beacon/states/head/validator_count?status=active",
				&validatorCountResponse,
			).Return(
				nil,
				test.validatorCountEndpointError,
			).SetArg(
				2,
				test.validatorCountResponse,
			).Times(test.validatorCountCalled)

			// Type assertion.
			var client iface.PrysmBeaconChainClient = &prysmBeaconChainClient{
				nodeClient:      &beaconApiNodeClient{jsonRestHandler: jsonRestHandler},
				jsonRestHandler: jsonRestHandler,
			}

			countResponse, err := client.GetValidatorCount(ctx, "head", []validator.ValidatorStatus{validator.Active})

			if len(test.expectedResponse) == 0 {
				require.ErrorContains(t, test.expectedError, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, test.expectedResponse, countResponse)
			}
		})
	}

}
