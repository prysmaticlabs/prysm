//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

// Check that the DomainData() returns whatever GetDomainData() returns
func TestBeaconApiValidatorClient_DomainDataValid(t *testing.T) {
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := params.BeaconConfig().DomainSyncCommittee[:]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedSignatureDomain := []byte{1, 2, 3, 4, 5}

	// Make sure that GetDomainData() is called exactly once and with the right arguments
	domainDataProvider := mock.NewMockdomainDataProvider(ctrl)
	domainDataProvider.EXPECT().GetDomainData(
		epoch,
		domainType,
	).Return(
		&ethpb.DomainResponse{SignatureDomain: expectedSignatureDomain},
		nil,
	).Times(1)

	validatorClient := beaconApiValidatorClient{domainDataProvider: domainDataProvider}
	resp, err := validatorClient.DomainData(context.Background(), &ethpb.DomainRequest{Epoch: epoch, Domain: domainType})
	assert.NoError(t, err)
	require.NotNil(t, resp)
	assert.DeepEqual(t, expectedSignatureDomain, resp.SignatureDomain)
}

// Check that the error that DomainData() returns contains the error returned by GetDomainData()
func TestBeaconApiValidatorClient_DomainDataError(t *testing.T) {
	epoch := params.BeaconConfig().AltairForkEpoch
	domainType := params.BeaconConfig().DomainSyncCommittee[:]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedError := errors.New("foo error")

	// Make sure that GetDomainData() is called exactly once and with the right arguments
	domainDataProvider := mock.NewMockdomainDataProvider(ctrl)
	domainDataProvider.EXPECT().GetDomainData(
		epoch,
		domainType,
	).Return(
		nil,
		expectedError,
	).Times(1)

	validatorClient := beaconApiValidatorClient{domainDataProvider: domainDataProvider}
	_, err := validatorClient.DomainData(context.Background(), &ethpb.DomainRequest{Epoch: epoch, Domain: domainType})
	require.ErrorIs(t, err, expectedError)
}
