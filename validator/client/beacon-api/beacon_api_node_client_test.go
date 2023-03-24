package beacon_api

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetGenesis(t *testing.T) {
	testCases := []struct {
		name                    string
		genesisResponse         *apimiddleware.GenesisResponse_GenesisJson
		genesisError            error
		depositContractResponse apimiddleware.DepositContractResponseJson
		depositContractError    error
		queriesDepositContract  bool
		expectedResponse        *ethpb.Genesis
		expectedError           string
	}{
		{
			name:          "fails to get genesis",
			genesisError:  errors.New("foo error"),
			expectedError: "failed to get genesis: foo error",
		},
		{
			name: "fails to decode genesis validator root",
			genesisResponse: &apimiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "1",
				GenesisValidatorsRoot: "foo",
			},
			expectedError: "failed to decode genesis validator root `foo`",
		},
		{
			name: "fails to parse genesis time",
			genesisResponse: &apimiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "foo",
				GenesisValidatorsRoot: hexutil.Encode([]byte{1}),
			},
			expectedError: "failed to parse genesis time `foo`",
		},
		{
			name: "fails to query contract information",
			genesisResponse: &apimiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "1",
				GenesisValidatorsRoot: hexutil.Encode([]byte{2}),
			},
			depositContractError:   errors.New("foo error"),
			queriesDepositContract: true,
			expectedError:          "failed to query deposit contract information: foo error",
		},
		{
			name: "fails to read nil deposit contract data",
			genesisResponse: &apimiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "1",
				GenesisValidatorsRoot: hexutil.Encode([]byte{2}),
			},
			queriesDepositContract: true,
			depositContractResponse: apimiddleware.DepositContractResponseJson{
				Data: nil,
			},
			expectedError: "deposit contract data is nil",
		},
		{
			name: "fails to decode deposit contract address",
			genesisResponse: &apimiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "1",
				GenesisValidatorsRoot: hexutil.Encode([]byte{2}),
			},
			queriesDepositContract: true,
			depositContractResponse: apimiddleware.DepositContractResponseJson{
				Data: &apimiddleware.DepositContractJson{
					Address: "foo",
				},
			},
			expectedError: "failed to decode deposit contract address `foo`",
		},
		{
			name: "successfully retrieves genesis info",
			genesisResponse: &apimiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "654812",
				GenesisValidatorsRoot: hexutil.Encode([]byte{2}),
			},
			queriesDepositContract: true,
			depositContractResponse: apimiddleware.DepositContractResponseJson{
				Data: &apimiddleware.DepositContractJson{
					Address: hexutil.Encode([]byte{3}),
				},
			},
			expectedResponse: &ethpb.Genesis{
				GenesisTime: &timestamppb.Timestamp{
					Seconds: 654812,
				},
				DepositContractAddress: []byte{3},
				GenesisValidatorsRoot:  []byte{2},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			genesisProvider := mock.NewMockgenesisProvider(ctrl)
			genesisProvider.EXPECT().GetGenesis(
				ctx,
			).Return(
				testCase.genesisResponse,
				nil,
				testCase.genesisError,
			)

			depositContractJson := apimiddleware.DepositContractResponseJson{}
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

			if testCase.queriesDepositContract {
				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					"/eth/v1/config/deposit_contract",
					&depositContractJson,
				).Return(
					nil,
					testCase.depositContractError,
				).SetArg(
					2,
					testCase.depositContractResponse,
				)
			}

			nodeClient := &beaconApiNodeClient{
				genesisProvider: genesisProvider,
				jsonRestHandler: jsonRestHandler,
			}
			response, err := nodeClient.GetGenesis(ctx, &emptypb.Empty{})

			if testCase.expectedResponse == nil {
				assert.ErrorContains(t, testCase.expectedError, err)
			} else {
				assert.DeepEqual(t, testCase.expectedResponse, response)
			}
		})
	}
}
