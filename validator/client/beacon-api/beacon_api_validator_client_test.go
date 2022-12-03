//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestBeaconApiValidatorClient_GetAttestationDataNilInput(t *testing.T) {
	validatorClient := beaconApiValidatorClient{}
	_, err := validatorClient.GetAttestationData(context.Background(), nil)
	assert.ErrorContains(t, "GetAttestationData received nil argument `in`", err)
}

// Make sure that GetAttestationData() returns the same thing as the internal getAttestationData()
func TestBeaconApiValidatorClient_GetAttestationDataValid(t *testing.T) {
	const slot = types.Slot(1)
	const committeeIndex = types.CommitteeIndex(2)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	produceAttestationDataResponseJson := rpcmiddleware.ProduceAttestationDataResponseJson{}
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("/eth/v1/validator/attestation_data?committee_index=%d&slot=%d", committeeIndex, slot),
		&produceAttestationDataResponseJson,
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		generateValidAttestation(uint64(slot), uint64(committeeIndex)),
	).Times(2)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	expectedResp, expectedErr := validatorClient.getAttestationData(slot, committeeIndex)

	resp, err := validatorClient.GetAttestationData(
		context.Background(),
		&ethpb.AttestationDataRequest{Slot: slot, CommitteeIndex: committeeIndex},
	)

	assert.DeepEqual(t, expectedErr, err)
	assert.DeepEqual(t, expectedResp, resp)
}

func TestBeaconApiValidatorClient_GetAttestationDataError(t *testing.T) {
	const slot = types.Slot(1)
	const committeeIndex = types.CommitteeIndex(2)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	produceAttestationDataResponseJson := rpcmiddleware.ProduceAttestationDataResponseJson{}
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("/eth/v1/validator/attestation_data?committee_index=%d&slot=%d", committeeIndex, slot),
		&produceAttestationDataResponseJson,
	).Return(
		nil,
		errors.New("some specific json error"),
	).SetArg(
		1,
		generateValidAttestation(uint64(slot), uint64(committeeIndex)),
	).Times(2)

	validatorClient := beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	expectedResp, expectedErr := validatorClient.getAttestationData(slot, committeeIndex)

	resp, err := validatorClient.GetAttestationData(
		context.Background(),
		&ethpb.AttestationDataRequest{Slot: slot, CommitteeIndex: committeeIndex},
	)

	assert.ErrorContains(t, expectedErr.Error(), err)
	assert.DeepEqual(t, expectedResp, resp)
}
