package beacon_api

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetGenesis(t *testing.T) {
	testCases := []struct {
		name                    string
		genesisResponse         *beacon.Genesis
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
			genesisResponse: &beacon.Genesis{
				GenesisTime:           "1",
				GenesisValidatorsRoot: "foo",
			},
			expectedError: "failed to decode genesis validator root `foo`",
		},
		{
			name: "fails to parse genesis time",
			genesisResponse: &beacon.Genesis{
				GenesisTime:           "foo",
				GenesisValidatorsRoot: hexutil.Encode([]byte{1}),
			},
			expectedError: "failed to parse genesis time `foo`",
		},
		{
			name: "fails to query contract information",
			genesisResponse: &beacon.Genesis{
				GenesisTime:           "1",
				GenesisValidatorsRoot: hexutil.Encode([]byte{2}),
			},
			depositContractError:   errors.New("foo error"),
			queriesDepositContract: true,
			expectedError:          "failed to query deposit contract information: foo error",
		},
		{
			name: "fails to read nil deposit contract data",
			genesisResponse: &beacon.Genesis{
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
			genesisResponse: &beacon.Genesis{
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
			genesisResponse: &beacon.Genesis{
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
				Data: &shared.SyncDetails{
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
				Data: &shared.SyncDetails{
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

func TestGetVersion(t *testing.T) {
	const versionEndpoint = "/eth/v1/node/version"

	testCases := []struct {
		name                 string
		restEndpointResponse apimiddleware.VersionResponseJson
		restEndpointError    error
		expectedResponse     *ethpb.Version
		expectedError        string
	}{
		{
			name:              "fails to query REST endpoint",
			restEndpointError: errors.New("foo error"),
			expectedError:     "failed to query node version",
		},
		{
			name:                 "returns nil version data",
			restEndpointResponse: apimiddleware.VersionResponseJson{Data: nil},
			expectedError:        "empty version response",
		},
		{
			name: "returns proper version response",
			restEndpointResponse: apimiddleware.VersionResponseJson{
				Data: &apimiddleware.VersionJson{
					Version: "prysm/local",
				},
			},
			expectedResponse: &ethpb.Version{
				Version: "prysm/local",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var versionResponse apimiddleware.VersionResponseJson
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				versionEndpoint,
				&versionResponse,
			).Return(
				nil,
				testCase.restEndpointError,
			).SetArg(
				2,
				testCase.restEndpointResponse,
			)

			nodeClient := &beaconApiNodeClient{jsonRestHandler: jsonRestHandler}
			version, err := nodeClient.GetVersion(ctx, &emptypb.Empty{})

			if testCase.expectedResponse == nil {
				assert.ErrorContains(t, testCase.expectedError, err)
			} else {
				assert.DeepEqual(t, testCase.expectedResponse, version)
			}
		})
	}
}
