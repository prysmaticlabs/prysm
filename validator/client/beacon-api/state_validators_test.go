package beacon_api

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestGetStateValidators_Nominal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := strings.Join([]string{
		"/eth/v1/beacon/states/head/validators?",
		"id=0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13&", // active_ongoing
		"id=0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526&", // active_exiting
		"id=0x424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242&", // does not exist
		"id=0x800015473bdc3a7f45ef8eb8abc598bc20021e55ad6e6ad1d745aaef9730dd2c28ec08bf42df18451de94dd4a6d24ec5",  // exited_slashed
	}, "")

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	wanted := []*rpcmiddleware.ValidatorContainerJson{
		{
			Index:  "55293",
			Status: "active_ongoing",
			Validator: &rpcmiddleware.ValidatorJson{
				PublicKey: "0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13",
			},
		},
		{
			Index:  "55294",
			Status: "active_exiting",
			Validator: &rpcmiddleware.ValidatorJson{
				PublicKey: "0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526",
			},
		},
		{
			Index:  "55295",
			Status: "exited_slashed",
			Validator: &rpcmiddleware.ValidatorJson{
				PublicKey: "0x800015473bdc3a7f45ef8eb8abc598bc20021e55ad6e6ad1d745aaef9730dd2c28ec08bf42df18451de94dd4a6d24ec5",
			},
		},
	}

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: wanted,
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	actual, err := validatorClient.getStateValidators([]string{
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing
		"0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526", // active_exiting
		"0x424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242", // does not exist
		"0x800015473bdc3a7f45ef8eb8abc598bc20021e55ad6e6ad1d745aaef9730dd2c28ec08bf42df18451de94dd4a6d24ec5", // exited_slashed
	})
	require.NoError(t, err)
	assert.DeepEqual(t, wanted, actual.Data)
}

func TestGetStateValidators_GetRestJsonResponseOnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := "/eth/v1/beacon/states/head/validators?id=0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13"

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		errors.New("an error"),
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	_, err := validatorClient.getStateValidators([]string{
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing
	})
	assert.ErrorContains(t, "an error", err)
	assert.ErrorContains(t, "failed to get json response", err)
}

func TestGetStateValidators_DataIsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := "/eth/v1/beacon/states/head/validators?id=0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13"

	stateValidatorsResponseJson := rpcmiddleware.StateValidatorsResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.StateValidatorsResponseJson{
			Data: nil,
		},
	).Times(1)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	_, err := validatorClient.getStateValidators([]string{
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing
	})
	assert.ErrorContains(t, "stateValidatorsJson.Data is nil", err)
}
