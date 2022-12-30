package beacon_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

const getAttesterDutiesTestEndpoint = "/eth/v1/validator/duties/attester"
const getProposerDutiesTestEndpoint = "/eth/v1/validator/duties/proposer"
const getSyncDutiesTestEndpoint = "/eth/v1/validator/duties/sync"
const getCommitteesTestEndpoint = "/eth/v1/beacon/states/head/committees"

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

func TestGetCommittees_Valid(t *testing.T) {
	const epoch = types.Epoch(1)

	expectedCommittees := apimiddleware.StateCommitteesResponseJson{
		Data: []*apimiddleware.CommitteeJson{
			{
				Index: "1",
				Slot:  "2",
				Validators: []string{
					"3",
					"4",
				},
			},
			{
				Index: "5",
				Slot:  "6",
				Validators: []string{
					"7",
					"8",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s?epoch=%d", getCommitteesTestEndpoint, epoch),
		&apimiddleware.StateCommitteesResponseJson{},
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		expectedCommittees,
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	committees, err := dutiesProvider.GetCommittees(epoch)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedCommittees.Data, committees)
}

func TestGetCommittees_HttpError(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s?epoch=%d", getCommitteesTestEndpoint, epoch),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetCommittees(epoch)
	assert.ErrorContains(t, "foo error", err)
	assert.ErrorContains(t, "failed to query committees for epoch `1`", err)
}

func TestGetCommittees_NilData(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s?epoch=%d", getCommitteesTestEndpoint, epoch),
		gomock.Any(),
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		apimiddleware.StateCommitteesResponseJson{
			Data: nil,
		},
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetCommittees(epoch)
	assert.ErrorContains(t, "state committees data is nil", err)
}

func TestGetCommittees_NilCommittee(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("%s?epoch=%d", getCommitteesTestEndpoint, epoch),
		gomock.Any(),
	).Return(
		nil,
		nil,
	).SetArg(
		1,
		apimiddleware.StateCommitteesResponseJson{
			Data: []*apimiddleware.CommitteeJson{nil},
		},
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetCommittees(epoch)
	assert.ErrorContains(t, "committee at index `0` is nil", err)
}

func TestGetDutiesForEpoch_Valid(t *testing.T) {
	testCases := []struct {
		name                string
		fetchSyncDuties     bool
		fetchProposerDuties bool
	}{
		{
			name:                "fetch attester duties",
			fetchSyncDuties:     false,
			fetchProposerDuties: false,
		},
		{
			name:                "fetch attester and sync duties",
			fetchSyncDuties:     true,
			fetchProposerDuties: false,
		},
		{
			name:                "fetch attester and proposer duties",
			fetchSyncDuties:     false,
			fetchProposerDuties: true,
		},
		{
			name:                "fetch attester and sync and proposer duties",
			fetchSyncDuties:     true,
			fetchProposerDuties: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			const epoch = types.Epoch(1)

			pubkeys := [][]byte{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}, {9}, {10}, {11}, {12}}
			validatorIndices := []types.ValidatorIndex{13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
			committeeIndices := []types.CommitteeIndex{25, 26, 27}
			committeeSlots := []types.Slot{28, 29, 30}
			proposerSlots := []types.Slot{31, 32, 33, 34, 35, 36, 37, 38}

			statuses := []ethpb.ValidatorStatus{
				ethpb.ValidatorStatus_UNKNOWN_STATUS,
				ethpb.ValidatorStatus_DEPOSITED,
				ethpb.ValidatorStatus_PENDING,
				ethpb.ValidatorStatus_ACTIVE,
				ethpb.ValidatorStatus_EXITING,
				ethpb.ValidatorStatus_SLASHING,
				ethpb.ValidatorStatus_EXITED,
				ethpb.ValidatorStatus_INVALID,
				ethpb.ValidatorStatus_PARTIALLY_DEPOSITED,
				ethpb.ValidatorStatus_UNKNOWN_STATUS,
				ethpb.ValidatorStatus_DEPOSITED,
				ethpb.ValidatorStatus_PENDING,
			}

			multipleValidatorStatus := &ethpb.MultipleValidatorStatusResponse{
				PublicKeys: pubkeys,
				Indices:    validatorIndices,
				Statuses: []*ethpb.ValidatorStatusResponse{
					{Status: statuses[0]},
					{Status: statuses[1]},
					{Status: statuses[2]},
					{Status: statuses[3]},
					{Status: statuses[4]},
					{Status: statuses[5]},
					{Status: statuses[6]},
					{Status: statuses[7]},
					{Status: statuses[8]},
					{Status: statuses[9]},
					{Status: statuses[10]},
					{Status: statuses[11]},
				},
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			dutiesProvider := mock.NewMockdutiesProvider(ctrl)
			dutiesProvider.EXPECT().GetCommittees(
				epoch,
			).Return(
				[]*apimiddleware.CommitteeJson{
					{
						Index: strconv.FormatUint(uint64(committeeIndices[0]), 10),
						Slot:  strconv.FormatUint(uint64(committeeSlots[0]), 10),
						Validators: []string{
							strconv.FormatUint(uint64(validatorIndices[0]), 10),
							strconv.FormatUint(uint64(validatorIndices[1]), 10),
						},
					},
					{
						Index: strconv.FormatUint(uint64(committeeIndices[1]), 10),
						Slot:  strconv.FormatUint(uint64(committeeSlots[1]), 10),
						Validators: []string{
							strconv.FormatUint(uint64(validatorIndices[2]), 10),
							strconv.FormatUint(uint64(validatorIndices[3]), 10),
						},
					},
					{
						Index: strconv.FormatUint(uint64(committeeIndices[2]), 10),
						Slot:  strconv.FormatUint(uint64(committeeSlots[2]), 10),
						Validators: []string{
							strconv.FormatUint(uint64(validatorIndices[4]), 10),
							strconv.FormatUint(uint64(validatorIndices[5]), 10),
						},
					},
				},
				nil,
			).Times(1)

			dutiesProvider.EXPECT().GetAttesterDuties(
				epoch,
				multipleValidatorStatus.Indices,
			).Return(
				[]*apimiddleware.AttesterDutyJson{
					{
						Pubkey:         hexutil.Encode(pubkeys[0]),
						ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[0]), 10),
						CommitteeIndex: strconv.FormatUint(uint64(committeeIndices[0]), 10),
						Slot:           strconv.FormatUint(uint64(committeeSlots[0]), 10),
					},
					{
						Pubkey:         hexutil.Encode(pubkeys[1]),
						ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[1]), 10),
						CommitteeIndex: strconv.FormatUint(uint64(committeeIndices[0]), 10),
						Slot:           strconv.FormatUint(uint64(committeeSlots[0]), 10),
					},
					{
						Pubkey:         hexutil.Encode(pubkeys[2]),
						ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[2]), 10),
						CommitteeIndex: strconv.FormatUint(uint64(committeeIndices[1]), 10),
						Slot:           strconv.FormatUint(uint64(committeeSlots[1]), 10),
					},
					{
						Pubkey:         hexutil.Encode(pubkeys[3]),
						ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[3]), 10),
						CommitteeIndex: strconv.FormatUint(uint64(committeeIndices[1]), 10),
						Slot:           strconv.FormatUint(uint64(committeeSlots[1]), 10),
					},
					{
						Pubkey:         hexutil.Encode(pubkeys[4]),
						ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[4]), 10),
						CommitteeIndex: strconv.FormatUint(uint64(committeeIndices[2]), 10),
						Slot:           strconv.FormatUint(uint64(committeeSlots[2]), 10),
					},
					{
						Pubkey:         hexutil.Encode(pubkeys[5]),
						ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[5]), 10),
						CommitteeIndex: strconv.FormatUint(uint64(committeeIndices[2]), 10),
						Slot:           strconv.FormatUint(uint64(committeeSlots[2]), 10),
					},
				},
				nil,
			).Times(1)

			if testCase.fetchProposerDuties {
				dutiesProvider.EXPECT().GetProposerDuties(
					epoch,
				).Return(
					[]*apimiddleware.ProposerDutyJson{
						{
							Pubkey:         hexutil.Encode(pubkeys[4]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[4]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[0]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[4]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[4]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[1]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[5]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[5]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[2]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[5]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[5]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[3]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[6]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[6]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[4]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[6]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[6]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[5]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[7]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[7]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[6]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[7]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[7]), 10),
							Slot:           strconv.FormatUint(uint64(proposerSlots[7]), 10),
						},
					},
					nil,
				).Times(1)
			}

			if testCase.fetchSyncDuties {
				dutiesProvider.EXPECT().GetSyncDuties(
					epoch,
					multipleValidatorStatus.Indices,
				).Return(
					[]*apimiddleware.SyncCommitteeDuty{
						{
							Pubkey:         hexutil.Encode(pubkeys[5]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[5]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[6]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[6]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[7]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[7]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[8]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[8]), 10),
						},
						{
							Pubkey:         hexutil.Encode(pubkeys[9]),
							ValidatorIndex: strconv.FormatUint(uint64(validatorIndices[9]), 10),
						},
					},
					nil,
				).Times(1)
			}

			var expectedProposerSlots1 []types.Slot
			var expectedProposerSlots2 []types.Slot
			var expectedProposerSlots3 []types.Slot
			var expectedProposerSlots4 []types.Slot

			if testCase.fetchProposerDuties {
				expectedProposerSlots1 = []types.Slot{
					proposerSlots[0],
					proposerSlots[1],
				}

				expectedProposerSlots2 = []types.Slot{
					proposerSlots[2],
					proposerSlots[3],
				}

				expectedProposerSlots3 = []types.Slot{
					proposerSlots[4],
					proposerSlots[5],
				}

				expectedProposerSlots4 = []types.Slot{
					proposerSlots[6],
					proposerSlots[7],
				}
			}

			expectedDuties := []*ethpb.DutiesResponse_Duty{
				{
					Committee: []types.ValidatorIndex{
						validatorIndices[0],
						validatorIndices[1],
					},
					CommitteeIndex: committeeIndices[0],
					AttesterSlot:   committeeSlots[0],
					PublicKey:      pubkeys[0],
					Status:         statuses[0],
					ValidatorIndex: validatorIndices[0],
				},
				{
					Committee: []types.ValidatorIndex{
						validatorIndices[0],
						validatorIndices[1],
					},
					CommitteeIndex: committeeIndices[0],
					AttesterSlot:   committeeSlots[0],
					PublicKey:      pubkeys[1],
					Status:         statuses[1],
					ValidatorIndex: validatorIndices[1],
				},
				{
					Committee: []types.ValidatorIndex{
						validatorIndices[2],
						validatorIndices[3],
					},
					CommitteeIndex: committeeIndices[1],
					AttesterSlot:   committeeSlots[1],
					PublicKey:      pubkeys[2],
					Status:         statuses[2],
					ValidatorIndex: validatorIndices[2],
				},
				{
					Committee: []types.ValidatorIndex{
						validatorIndices[2],
						validatorIndices[3],
					},
					CommitteeIndex: committeeIndices[1],
					AttesterSlot:   committeeSlots[1],
					PublicKey:      pubkeys[3],
					Status:         statuses[3],
					ValidatorIndex: validatorIndices[3],
				},
				{
					Committee: []types.ValidatorIndex{
						validatorIndices[4],
						validatorIndices[5],
					},
					CommitteeIndex: committeeIndices[2],
					AttesterSlot:   committeeSlots[2],
					PublicKey:      pubkeys[4],
					Status:         statuses[4],
					ValidatorIndex: validatorIndices[4],
					ProposerSlots:  expectedProposerSlots1,
				},
				{
					Committee: []types.ValidatorIndex{
						validatorIndices[4],
						validatorIndices[5],
					},
					CommitteeIndex:  committeeIndices[2],
					AttesterSlot:    committeeSlots[2],
					PublicKey:       pubkeys[5],
					Status:          statuses[5],
					ValidatorIndex:  validatorIndices[5],
					ProposerSlots:   expectedProposerSlots2,
					IsSyncCommittee: testCase.fetchSyncDuties,
				},
				{
					PublicKey:       pubkeys[6],
					Status:          statuses[6],
					ValidatorIndex:  validatorIndices[6],
					ProposerSlots:   expectedProposerSlots3,
					IsSyncCommittee: testCase.fetchSyncDuties,
				},
				{
					PublicKey:       pubkeys[7],
					Status:          statuses[7],
					ValidatorIndex:  validatorIndices[7],
					ProposerSlots:   expectedProposerSlots4,
					IsSyncCommittee: testCase.fetchSyncDuties,
				},
				{
					PublicKey:       pubkeys[8],
					Status:          statuses[8],
					ValidatorIndex:  validatorIndices[8],
					IsSyncCommittee: testCase.fetchSyncDuties,
				},
				{
					PublicKey:       pubkeys[9],
					Status:          statuses[9],
					ValidatorIndex:  validatorIndices[9],
					IsSyncCommittee: testCase.fetchSyncDuties,
				},
				{
					PublicKey:      pubkeys[10],
					Status:         statuses[10],
					ValidatorIndex: validatorIndices[10],
				},
				{
					PublicKey:      pubkeys[11],
					Status:         statuses[11],
					ValidatorIndex: validatorIndices[11],
				},
			}

			validatorClient := &beaconApiValidatorClient{dutiesProvider: dutiesProvider}
			duties, err := validatorClient.getDutiesForEpoch(
				epoch,
				multipleValidatorStatus,
				testCase.fetchProposerDuties,
				testCase.fetchSyncDuties,
			)
			require.NoError(t, err)
			assert.DeepEqual(t, expectedDuties, duties)
		})
	}
}
