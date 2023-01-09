package beacon_api

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestGetSyncMessageBlockRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const blockRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	tests := []struct {
		name                 string
		endpointError        error
		expectedErrorMessage string
		expectedResponse     apimiddleware.BlockRootResponseJson
	}{
		{
			name: "valid request",
			expectedResponse: apimiddleware.BlockRootResponseJson{
				Data: &apimiddleware.BlockRootContainerJson{
					Root: blockRoot,
				},
			},
		},
		{
			name:                 "internal server error",
			expectedErrorMessage: "internal server error",
			endpointError:        errors.New("internal server error"),
		},
		{
			name: "execution optimistic",
			expectedResponse: apimiddleware.BlockRootResponseJson{
				ExecutionOptimistic: true,
			},
			expectedErrorMessage: "the node is currently optimistic and cannot serve validators",
		},
		{
			name:                 "no data",
			expectedResponse:     apimiddleware.BlockRootResponseJson{},
			expectedErrorMessage: "no data returned",
		},
		{
			name: "no root",
			expectedResponse: apimiddleware.BlockRootResponseJson{
				Data: new(apimiddleware.BlockRootContainerJson),
			},
			expectedErrorMessage: "no root returned",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				"/eth/v1/beacon/blocks/head/root",
				&apimiddleware.BlockRootResponseJson{},
			).SetArg(
				2,
				test.expectedResponse,
			).Return(
				nil,
				test.endpointError,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			actualResponse, err := validatorClient.getSyncMessageBlockRoot(ctx)
			if test.expectedErrorMessage != "" {
				require.ErrorContains(t, test.expectedErrorMessage, err)
				return
			}

			require.NoError(t, err)

			expectedRootBytes, err := hexutil.Decode(test.expectedResponse.Data.Root)
			require.NoError(t, err)

			require.NoError(t, err)
			require.DeepEqual(t, expectedRootBytes, actualResponse.Root)
		})
	}
}
