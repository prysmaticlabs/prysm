//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestBeaconApiValidatorClient_GetAttestationDataNilInput(t *testing.T) {
	validatorClient := beaconApiValidatorClient{}
	_, err := validatorClient.GetAttestationData(context.Background(), nil)
	assert.ErrorContains(t, "GetAttestationData received nil argument `in`", err)
}

func TestBeaconApiValidatorClient_GetAttestationDataValid(t *testing.T) {
	slot := types.Slot(1)
	committeeIndex := types.CommitteeIndex(2)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	beaconBlockRoot, err := hexutil.Decode("0x7301ba3b16a5779e179b005f968bb6bda8bd7ffa302ebf6b0106fc4c6910f0f0")
	require.NoError(t, err)

	sourceRoot, err := hexutil.Decode("0xf7fb3f8571d9d429af3160e11f0c3cbbaa4b611c43ce4b3865e3935b9d16ac79")
	require.NoError(t, err)

	targetRoot, err := hexutil.Decode("0x3e3b613c12ea5af8cb421b2ce4d792d32129e5d649d97dc92d65bdce5c8e20d3")
	require.NoError(t, err)

	expectedAttestation := &ethpb.AttestationData{
		Slot:            1,
		CommitteeIndex:  2,
		BeaconBlockRoot: beaconBlockRoot,
		Source: &ethpb.Checkpoint{
			Epoch: 3,
			Root:  sourceRoot,
		},
		Target: &ethpb.Checkpoint{
			Epoch: 4,
			Root:  targetRoot,
		},
	}

	// Make sure that GetDomainData() is called exactly once and with the right arguments
	attestationDataProvider := mock.NewMockattestationDataProvider(ctrl)
	attestationDataProvider.EXPECT().GetAttestationData(
		slot,
		committeeIndex,
	).Return(
		expectedAttestation,
		nil,
	).Times(1)

	validatorClient := beaconApiValidatorClient{attestationDataProvider: attestationDataProvider}
	resp, err := validatorClient.GetAttestationData(
		context.Background(),
		&ethpb.AttestationDataRequest{Slot: slot, CommitteeIndex: committeeIndex},
	)
	assert.NoError(t, err)
	assert.DeepEqual(t, expectedAttestation, resp)
}

func TestBeaconApiValidatorClient_GetAttestationDataError(t *testing.T) {
	slot := types.Slot(1)
	committeeIndex := types.CommitteeIndex(2)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedError := errors.New("foo error")

	// Make sure that GetDomainData() is called exactly once and with the right arguments
	attestationDataProvider := mock.NewMockattestationDataProvider(ctrl)
	attestationDataProvider.EXPECT().GetAttestationData(
		slot,
		committeeIndex,
	).Return(
		nil,
		expectedError,
	).Times(1)

	validatorClient := beaconApiValidatorClient{attestationDataProvider: attestationDataProvider}
	_, err := validatorClient.GetAttestationData(
		context.Background(),
		&ethpb.AttestationDataRequest{Slot: slot, CommitteeIndex: committeeIndex},
	)
	require.ErrorIs(t, err, expectedError)
}
