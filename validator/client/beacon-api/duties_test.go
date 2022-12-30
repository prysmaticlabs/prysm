package beacon_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

const getAttesterDutiesTestEndpoint = "/eth/v1/validator/duties/attester"
const getProposerDutiesTestEndpoint = "/eth/v1/validator/duties/proposer"
const getSyncDutiesTestEndpoint = "/eth/v1/validator/duties/sync"

func TestGetAttesterDuties_Valid(t *testing.T) {
	stringValidatorIndices := []string{"2", "9"}
	const epoch = types.Epoch(1)

	validatorIndicesBytes, err := json.Marshal(stringValidatorIndices)
	require.NoError(t, err)

	expectedAttesterDuties := apimiddleware.AttesterDutiesResponseJson{
		Data: []*apimiddleware.AttesterDutyJson{
			{
				Pubkey:                  hexutil.Encode([]byte{1}),
				ValidatorIndex:          "2",
				CommitteeIndex:          "3",
				CommitteeLength:         "4",
				CommitteesAtSlot:        "5",
				ValidatorCommitteeIndex: "6",
				Slot:                    "7",
			},
			{
				Pubkey:                  hexutil.Encode([]byte{8}),
				ValidatorIndex:          "9",
				CommitteeIndex:          "10",
				CommitteeLength:         "11",
				CommitteesAtSlot:        "12",
				ValidatorCommitteeIndex: "13",
				Slot:                    "14",
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	validatorIndices := []types.ValidatorIndex{2, 9}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		fmt.Sprintf("%s/%d", getAttesterDutiesTestEndpoint, epoch),
		nil,
		bytes.NewBuffer(validatorIndicesBytes),
		&apimiddleware.AttesterDutiesResponseJson{},
	).Return(
		nil,
		nil,
	).SetArg(
		3,
		expectedAttesterDuties,
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	attesterDuties, err := dutiesProvider.GetAttesterDuties(epoch, validatorIndices)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedAttesterDuties.Data, attesterDuties)
}

func TestGetAttesterDuties_HttpError(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		fmt.Sprintf("%s/%d", getAttesterDutiesTestEndpoint, epoch),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetAttesterDuties(epoch, nil)
	assert.ErrorContains(t, "foo error", err)
	assert.ErrorContains(t, "failed to send POST data to REST endpoint", err)
}

func TestGetAttesterDuties_NilAttesterDuty(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		fmt.Sprintf("%s/%d", getAttesterDutiesTestEndpoint, epoch),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		nil,
	).SetArg(
		3,
		apimiddleware.AttesterDutiesResponseJson{
			Data: []*apimiddleware.AttesterDutyJson{nil},
		},
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetAttesterDuties(epoch, nil)
	assert.ErrorContains(t, "attester duty at index `0` is nil", err)
}

func TestGetProposerDuties_Valid(t *testing.T) {
	const epoch = types.Epoch(1)

	expectedProposerDuties := apimiddleware.ProposerDutiesResponseJson{
		Data: []*apimiddleware.ProposerDutyJson{
			{
				Pubkey:         hexutil.Encode([]byte{1}),
				ValidatorIndex: "2",
				Slot:           "3",
			},
			{
				Pubkey:         hexutil.Encode([]byte{4}),
				ValidatorIndex: "5",
				Slot:           "6",
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s/%d", getProposerDutiesTestEndpoint, epoch),
		&apimiddleware.ProposerDutiesResponseJson{},
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		expectedProposerDuties,
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	proposerDuties, err := dutiesProvider.GetProposerDuties(epoch)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedProposerDuties.Data, proposerDuties)
}

func TestGetProposerDuties_HttpError(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s/%d", getProposerDutiesTestEndpoint, epoch),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetProposerDuties(epoch)
	assert.ErrorContains(t, "foo error", err)
	assert.ErrorContains(t, "failed to query proposer duties for epoch `1`", err)
}

func TestGetProposerDuties_NilProposerDuty(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s/%d", getProposerDutiesTestEndpoint, epoch),
		gomock.Any(),
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		apimiddleware.ProposerDutiesResponseJson{
			Data: []*apimiddleware.ProposerDutyJson{nil},
		},
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetProposerDuties(epoch)
	assert.ErrorContains(t, "proposer duty at index `0` is nil", err)
}

func TestGetSyncDuties_Valid(t *testing.T) {
	stringValidatorIndices := []string{"2", "6"}
	const epoch = types.Epoch(1)

	validatorIndicesBytes, err := json.Marshal(stringValidatorIndices)
	require.NoError(t, err)

	expectedSyncDuties := apimiddleware.SyncCommitteeDutiesResponseJson{
		Data: []*apimiddleware.SyncCommitteeDuty{
			{
				Pubkey:         hexutil.Encode([]byte{1}),
				ValidatorIndex: "2",
				ValidatorSyncCommitteeIndices: []string{
					"3",
					"4",
				},
			},
			{
				Pubkey:         hexutil.Encode([]byte{5}),
				ValidatorIndex: "6",
				ValidatorSyncCommitteeIndices: []string{
					"7",
					"8",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	validatorIndices := []types.ValidatorIndex{2, 6}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		fmt.Sprintf("%s/%d", getSyncDutiesTestEndpoint, epoch),
		nil,
		bytes.NewBuffer(validatorIndicesBytes),
		&apimiddleware.SyncCommitteeDutiesResponseJson{},
	).Return(
		nil,
		nil,
	).SetArg(
		3,
		expectedSyncDuties,
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	syncDuties, err := dutiesProvider.GetSyncDuties(epoch, validatorIndices)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedSyncDuties.Data, syncDuties)
}

func TestGetSyncDuties_HttpError(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		fmt.Sprintf("%s/%d", getSyncDutiesTestEndpoint, epoch),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetSyncDuties(epoch, nil)
	assert.ErrorContains(t, "foo error", err)
	assert.ErrorContains(t, "failed to send POST data to REST endpoint", err)
}

func TestGetSyncDuties_NilSyncDuty(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		fmt.Sprintf("%s/%d", getSyncDutiesTestEndpoint, epoch),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		nil,
	).SetArg(
		3,
		apimiddleware.SyncCommitteeDutiesResponseJson{
			Data: []*apimiddleware.SyncCommitteeDuty{nil},
		},
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetSyncDuties(epoch, nil)
	assert.ErrorContains(t, "sync duty at index `0` is nil", err)
}
