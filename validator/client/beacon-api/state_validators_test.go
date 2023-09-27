package beacon_api

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
)

func TestGetStateValidators_Nominal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := strings.Join([]string{
		"/eth/v1/beacon/states/head/validators?",
		"id=12345&",
		"id=0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13&", // active_ongoing
		"id=0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526&", // active_exiting
		"id=0x424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242&", // does not exist
		"id=0x800015473bdc3a7f45ef8eb8abc598bc20021e55ad6e6ad1d745aaef9730dd2c28ec08bf42df18451de94dd4a6d24ec5&", // exited_slashed
		"status=active_ongoing&status=active_exiting&status=exited_slashed&status=exited_unslashed",
	}, "")

	stateValidatorsResponseJson := beacon.GetValidatorsResponse{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	wanted := []*beacon.ValidatorContainer{
		{
			Index:  "12345",
			Status: "active_ongoing",
			Validator: &beacon.Validator{
				Pubkey: "0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be19",
			},
		},
		{
			Index:  "55293",
			Status: "active_ongoing",
			Validator: &beacon.Validator{
				Pubkey: "0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13",
			},
		},
		{
			Index:  "55294",
			Status: "active_exiting",
			Validator: &beacon.Validator{
				Pubkey: "0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526",
			},
		},
		{
			Index:  "55295",
			Status: "exited_slashed",
			Validator: &beacon.Validator{
				Pubkey: "0x800015473bdc3a7f45ef8eb8abc598bc20021e55ad6e6ad1d745aaef9730dd2c28ec08bf42df18451de94dd4a6d24ec5",
			},
		},
	}

	ctx := context.Background()

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		beacon.GetValidatorsResponse{
			Data: wanted,
		},
	).Times(1)

	stateValidatorsProvider := beaconApiStateValidatorsProvider{jsonRestHandler: jsonRestHandler}
	actual, err := stateValidatorsProvider.GetStateValidators(ctx, []string{
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing
		"0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526", // active_exiting
		"0x424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242424242", // does not exist
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing - duplicate
		"0x800015473bdc3a7f45ef8eb8abc598bc20021e55ad6e6ad1d745aaef9730dd2c28ec08bf42df18451de94dd4a6d24ec5", // exited_slashed
	},
		[]int64{
			12345, // active_ongoing
			12345, // active_ongoing - duplicate
		},
		[]string{"active_ongoing", "active_exiting", "exited_slashed", "exited_unslashed"},
	)
	require.NoError(t, err)
	assert.DeepEqual(t, wanted, actual.Data)
}

func TestGetStateValidators_GetRestJsonResponseOnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := "/eth/v1/beacon/states/head/validators?id=0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13"

	stateValidatorsResponseJson := beacon.GetValidatorsResponse{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	ctx := context.Background()

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		errors.New("an error"),
	).Times(1)

	stateValidatorsProvider := beaconApiStateValidatorsProvider{jsonRestHandler: jsonRestHandler}
	_, err := stateValidatorsProvider.GetStateValidators(ctx, []string{
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing
	},
		nil,
		nil,
	)
	assert.ErrorContains(t, "an error", err)
	assert.ErrorContains(t, "failed to get json response", err)
}

func TestGetStateValidators_DataIsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	url := "/eth/v1/beacon/states/head/validators?id=0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13"

	ctx := context.Background()
	stateValidatorsResponseJson := beacon.GetValidatorsResponse{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		url,
		&stateValidatorsResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		beacon.GetValidatorsResponse{
			Data: nil,
		},
	).Times(1)

	stateValidatorsProvider := beaconApiStateValidatorsProvider{jsonRestHandler: jsonRestHandler}
	_, err := stateValidatorsProvider.GetStateValidators(ctx, []string{
		"0x8000091c2ae64ee414a54c1cc1fc67dec663408bc636cb86756e0200e41a75c8f86603f104f02c856983d2783116be13", // active_ongoing
	},
		nil,
		nil,
	)
	assert.ErrorContains(t, "stateValidatorsJson.Data is nil", err)
}
