package beacon_api

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

const defaultStringPubKey = "0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13"

func TestValidatorStatus_NotActive(t *testing.T) {
	pubKey, err := hexutil.Decode(defaultStringPubKey)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("/eth/v1/beacon/states/head/validators?id=%s", stringPubKey),
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:  "55300",
					Status: "pending_queued",
					Validator: &rpcmiddleware.ValidatorJson{
						ActivationEpoch: "100",
					},
				},
			},
		},
	).Times(1)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		"/eth/v1/beacon/states/head/validators?status=active",
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:     "55293",
					Status:    "active_ongoing",
					Validator: &rpcmiddleware.ValidatorJson{},
				},
				{
					Index:     "55294",
					Status:    "active_ongoing",
					Validator: &rpcmiddleware.ValidatorJson{},
				},
			},
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}

	actual, err := validatorClient.validatorStatus(
		&ethpb.ValidatorStatusRequest{
			PublicKey: pubKey,
		},
	)
	require.NoError(t, err)

	wanted := &ethpb.ValidatorStatusResponse{
		Status:                    ethpb.ValidatorStatus_PENDING,
		ActivationEpoch:           100,
		PositionInActivationQueue: 6,
	}

	assert.DeepEqual(t, wanted, actual)
}

func TestValidatorStatus_UnknownStatus(t *testing.T) {
	pubKey, err := hexutil.Decode(defaultStringPubKey)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		gomock.Any(),
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{},
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}

	actual, err := validatorClient.ValidatorStatus(
		context.Background(),
		&ethpb.ValidatorStatusRequest{
			PublicKey: pubKey,
		},
	)
	require.NoError(t, err)

	wanted := &ethpb.ValidatorStatusResponse{
		Status:          ethpb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	assert.DeepEqual(t, wanted, actual)
}

func TestValidatorStatus_InvalidData(t *testing.T) {
	testCases := []struct {
		name                 string
		data                 []*rpcmiddleware.ValidatorContainerJson
		expectedErrorMessage string
		err                  error
	}{
		{
			name:                 "bad validator status",
			data:                 []*rpcmiddleware.ValidatorContainerJson{},
			expectedErrorMessage: "failed to get validator state",
			err:                  errors.New("some specific json error"),
		},
		{
			name: "bad validator status",
			data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:  "12345",
					Status: "NotAStatus",
					Validator: &rpcmiddleware.ValidatorJson{
						PublicKey: stringPubKey,
					},
				},
			},
			expectedErrorMessage: "invalid validator status: NotAStatus",
			err:                  nil,
		},
		{
			name: "bad activation epoch",
			data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Status: "pending_queued",
					Validator: &rpcmiddleware.ValidatorJson{
						ActivationEpoch: "NotAnEpoch",
					},
				},
			},
			expectedErrorMessage: "failed to parse activation epoch",
			err:                  nil,
		},
		{
			name: "bad validator index",
			data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Status: "pending_queued",
					Validator: &rpcmiddleware.ValidatorJson{
						ActivationEpoch: "12345",
					},
					Index: "NotAnIndex",
				},
			},
			expectedErrorMessage: "failed to parse validator index",
			err:                  nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name,
			func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					gomock.Any(),
					gomock.Any(),
				).Return(
					nil,
					testCase.err,
				).SetArg(
					1,
					rpcmiddleware.StateValidatorsResponseJson{
						Data: testCase.data,
					},
				).Times(1)

				validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}

				_, err := validatorClient.ValidatorStatus(
					context.Background(),
					&ethpb.ValidatorStatusRequest{},
				)
				assert.ErrorContains(t, testCase.expectedErrorMessage, err)
			},
		)
	}
}

func TestValidatorStatus_InvalidValidatorsState(t *testing.T) {
	pubKey, err := hexutil.Decode(defaultStringPubKey)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		gomock.Any(),
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:  "55300",
					Status: "pending_queued",
					Validator: &rpcmiddleware.ValidatorJson{
						ActivationEpoch: "100",
					},
				},
			},
		},
	).Times(1)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		"/eth/v1/beacon/states/head/validators?status=active",
		&stateValidatorsResponseJson,
	).Return(
		nil,
		errors.New("a specific error"),
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}

	_, err = validatorClient.validatorStatus(
		&ethpb.ValidatorStatusRequest{
			PublicKey: pubKey,
		},
	)
	require.ErrorContains(t, "failed to get validators state", err)
}

func TestValidatorStatus_InvalidLastValidatorIndex(t *testing.T) {
	pubKey, err := hexutil.Decode(defaultStringPubKey)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		gomock.Any(),
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:  "55300",
					Status: "pending_queued",
					Validator: &rpcmiddleware.ValidatorJson{
						ActivationEpoch: "100",
					},
				},
			},
		},
	).Times(1)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		"/eth/v1/beacon/states/head/validators?status=active",
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: []*rpcmiddleware.ValidatorContainerJson{
				{
					Index:     "NotAnIndex",
					Status:    "active_ongoing",
					Validator: &rpcmiddleware.ValidatorJson{},
				},
			},
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}

	_, err = validatorClient.validatorStatus(
		&ethpb.ValidatorStatusRequest{
			PublicKey: pubKey,
		},
	)
	require.ErrorContains(t, "failed to parse last validator index", err)
}
