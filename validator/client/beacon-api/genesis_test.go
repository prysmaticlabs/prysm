package beacon_api

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestGetGenesis_ValidGenesis(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	genesisResponseJson := rpcmiddleware.GenesisResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		"/eth/v1/beacon/genesis",
		&genesisResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.GenesisResponseJson{
			Data: &rpcmiddleware.GenesisResponse_GenesisJson{
				GenesisTime:           "1234",
				GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			},
		},
	).Times(1)

	genesisProvider := &beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler}
	resp, httpError, err := genesisProvider.GetGenesis()
	assert.NoError(t, err)
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	require.NotNil(t, resp)
	assert.Equal(t, "1234", resp.GenesisTime)
	assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", resp.GenesisValidatorsRoot)
}

func TestGetGenesis_NilData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	genesisResponseJson := rpcmiddleware.GenesisResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		"/eth/v1/beacon/genesis",
		&genesisResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		rpcmiddleware.GenesisResponseJson{Data: nil},
	).Times(1)

	genesisProvider := &beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler}
	_, httpError, err := genesisProvider.GetGenesis()
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	assert.ErrorContains(t, "genesis data is nil", err)
}

func TestGetGenesis_JsonResponseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedHttpErrorJson := &apimiddleware.DefaultErrorJson{
		Message: "http error message",
		Code:    999,
	}

	genesisResponseJson := rpcmiddleware.GenesisResponseJson{}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		"/eth/v1/beacon/genesis",
		&genesisResponseJson,
	).Return(
		expectedHttpErrorJson,
		errors.New("some specific json response error"),
	).Times(1)

	genesisProvider := &beaconApiGenesisProvider{jsonRestHandler: jsonRestHandler}
	_, httpError, err := genesisProvider.GetGenesis()
	assert.ErrorContains(t, "failed to get json response", err)
	assert.ErrorContains(t, "some specific json response error", err)
	assert.DeepEqual(t, expectedHttpErrorJson, httpError)
}
