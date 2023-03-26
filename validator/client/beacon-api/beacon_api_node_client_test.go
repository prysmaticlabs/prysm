package beacon_api

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetSyncStatus(t *testing.T) {
	const syncingEndpoint = "/eth/v1/node/syncing"

	testCases := []struct {
		name                 string
		restEndpointResponse apimiddleware.SyncingResponseJson
		restEndpointError    error
		expectedResponse     *ethpb.SyncStatus
		expectedError        string
	}{
		{
			name:              "fails to query REST endpoint",
			restEndpointError: errors.New("foo error"),
			expectedError:     "failed to get sync status: foo error",
		},
		{
			name:                 "returns nil syncing data",
			restEndpointResponse: apimiddleware.SyncingResponseJson{Data: nil},
			expectedError:        "syncing data is nil",
		},
		{
			name: "returns false syncing status",
			restEndpointResponse: apimiddleware.SyncingResponseJson{
				Data: &helpers.SyncDetailsJson{
					IsSyncing: false,
				},
			},
			expectedResponse: &ethpb.SyncStatus{
				Syncing: false,
			},
		},
		{
			name: "returns true syncing status",
			restEndpointResponse: apimiddleware.SyncingResponseJson{
				Data: &helpers.SyncDetailsJson{
					IsSyncing: true,
				},
			},
			expectedResponse: &ethpb.SyncStatus{
				Syncing: true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			syncingResponse := apimiddleware.SyncingResponseJson{}
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				syncingEndpoint,
				&syncingResponse,
			).Return(
				nil,
				testCase.restEndpointError,
			).SetArg(
				2,
				testCase.restEndpointResponse,
			)

			nodeClient := &beaconApiNodeClient{jsonRestHandler: jsonRestHandler}
			syncStatus, err := nodeClient.GetSyncStatus(ctx, &emptypb.Empty{})

			if testCase.expectedResponse == nil {
				assert.ErrorContains(t, testCase.expectedError, err)
			} else {
				assert.DeepEqual(t, testCase.expectedResponse, syncStatus)
			}
		})
	}

}
