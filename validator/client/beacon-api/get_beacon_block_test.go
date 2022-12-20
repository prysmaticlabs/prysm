package beacon_api

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestGetBeaconBlock_RequestFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	_, err := validatorClient.getBeaconBlock(1, []byte{1}, []byte{2})
	assert.ErrorContains(t, "failed to query GET REST endpoint", err)
	assert.ErrorContains(t, "foo error", err)
}

func TestGetBeaconBlock_Error(t *testing.T) {
	phase0BeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockJson{})
	require.NoError(t, err)
	altairBeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockAltairJson{})
	require.NoError(t, err)
	bellatrixBeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockBellatrixJson{})
	require.NoError(t, err)

	testCases := []struct {
		name                 string
		beaconBlock          interface{}
		expectedErrorMessage string
		consensusVersion     string
		data                 json.RawMessage
	}{
		{
			name:                 "phase0 block decoding failed",
			expectedErrorMessage: "failed to decode phase0 block response json",
			consensusVersion:     "phase0",
			data:                 []byte{},
		},
		{
			name:                 "phase0 block conversion failed",
			expectedErrorMessage: "failed to get phase0 block",
			consensusVersion:     "phase0",
			data:                 phase0BeaconBlockBytes,
		},
		{
			name:                 "altair block decoding failed",
			expectedErrorMessage: "failed to decode altair block response json",
			consensusVersion:     "altair",
			data:                 []byte{},
		},
		{
			name:                 "altair block conversion failed",
			expectedErrorMessage: "failed to get altair block",
			consensusVersion:     "altair",
			data:                 altairBeaconBlockBytes,
		},
		{
			name:                 "bellatrix block decoding failed",
			expectedErrorMessage: "failed to decode bellatrix block response json",
			beaconBlock:          "foo",
			consensusVersion:     "bellatrix",
			data:                 []byte{},
		},
		{
			name:                 "bellatrix block conversion failed",
			expectedErrorMessage: "failed to get bellatrix block",
			consensusVersion:     "bellatrix",
			data:                 bellatrixBeaconBlockBytes,
		},
		{
			name:                 "unsupported consensus version",
			expectedErrorMessage: "unsupported consensus version `foo`",
			consensusVersion:     "foo",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				gomock.Any(),
				&abstractProduceBlockResponseJson{},
			).SetArg(
				1,
				abstractProduceBlockResponseJson{
					Version: testCase.consensusVersion,
					Data:    testCase.data,
				},
			).Return(
				nil,
				nil,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			_, err := validatorClient.getBeaconBlock(1, []byte{1}, []byte{2})
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}
