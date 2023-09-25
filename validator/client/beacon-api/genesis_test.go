package beacon_api

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
)

func TestGetGenesis_ValidGenesis(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	genesisResponseJson := beacon.GetGenesisResponse{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		"/eth/v1/beacon/genesis",
		&genesisResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		beacon.GetGenesisResponse{
			Data: &beacon.Genesis{
				GenesisTime:           "1234",
				GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			},
		},
	).Times(1)

	genesisProvider := &beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler}
	resp, httpError, err := genesisProvider.GetGenesis(ctx)
	assert.NoError(t, err)
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	require.NotNil(t, resp)
	assert.Equal(t, "1234", resp.GenesisTime)
	assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", resp.GenesisValidatorsRoot)
}

func TestGetGenesis_NilData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	genesisResponseJson := beacon.GetGenesisResponse{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		"/eth/v1/beacon/genesis",
		&genesisResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		2,
		beacon.GetGenesisResponse{Data: nil},
	).Times(1)

	genesisProvider := &beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler}
	_, httpError, err := genesisProvider.GetGenesis(ctx)
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	assert.ErrorContains(t, "genesis data is nil", err)
}

func TestGetGenesis_JsonResponseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	expectedHttpErrorJson := &apimiddleware.DefaultErrorJson{
		Message: "http error message",
		Code:    999,
	}

	genesisResponseJson := beacon.GetGenesisResponse{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		"/eth/v1/beacon/genesis",
		&genesisResponseJson,
	).Return(
		expectedHttpErrorJson,
		errors.New("some specific json response error"),
	).Times(1)

	genesisProvider := &beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler}
	_, httpError, err := genesisProvider.GetGenesis(ctx)
	assert.ErrorContains(t, "failed to get json response", err)
	assert.ErrorContains(t, "some specific json response error", err)
	assert.DeepEqual(t, expectedHttpErrorJson, httpError)
}
