package beacon_api

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	apimiddleware2 "github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
	"testing"
)

func TestGetSyncMessageBlockRoot_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const blockRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	tests := []struct {
		name                 string
		expectedErrorMessage string
		errorJson            apimiddleware2.DefaultErrorJson
		expectedResponse     apimiddleware.BlockRootResponseJson
	}{
		{
			name: "valid request",
			errorJson: apimiddleware2.DefaultErrorJson{
				Code: 200,
			},
			expectedResponse: apimiddleware.BlockRootResponseJson{
				Data: &apimiddleware.BlockRootContainerJson{
					Root: blockRoot,
				},
			},
		},
		{
			name: "internal server error",
			errorJson: apimiddleware2.DefaultErrorJson{
				Message: "Internal server error",
				Code:    500,
			},
			expectedErrorMessage: "get request failed with status code: 500 and message: Internal server error",
		},
		{
			name: "execution optimistic",
			errorJson: apimiddleware2.DefaultErrorJson{
				Code: 200,
			},
			expectedResponse: apimiddleware.BlockRootResponseJson{
				ExecutionOptimistic: true,
			},
			expectedErrorMessage: "the node is currently optimistic and cannot serve validators",
		},
		{
			name: "block not found",
			errorJson: apimiddleware2.DefaultErrorJson{
				Message: "Block not found",
				Code:    404,
			},
			expectedErrorMessage: "get request failed with status code: 404 and message: Block not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				"/eth/v1/beacon/blocks/head/root",
				&apimiddleware.BlockRootResponseJson{},
			).SetArg(
				1,
				test.expectedResponse,
			).Return(
				&test.errorJson,
				nil,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			actualResponse, err := validatorClient.getSyncMessageBlockRoot()
			if test.expectedErrorMessage != "" {
				require.ErrorContains(t, test.expectedErrorMessage, err)
				return
			}

			expectedRootBytes, err := hexutil.Decode(test.expectedResponse.Data.Root)
			require.NoError(t, err)

			require.NoError(t, err)
			require.DeepEqual(t, expectedRootBytes, actualResponse.Root)
		})
	}
}
