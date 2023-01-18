package beacon_api

import (
	"bytes"
	"context"
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

	ctx := context.Background()

	validatorIndices := []types.ValidatorIndex{2, 9}
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		ctx,
		fmt.Sprintf("%s/%d", getAttesterDutiesTestEndpoint, epoch),
		nil,
		bytes.NewBuffer(validatorIndicesBytes),
		&apimiddleware.AttesterDutiesResponseJson{},
	).Return(
		nil,
		nil,
	).SetArg(
		4,
		expectedAttesterDuties,
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	attesterDuties, err := dutiesProvider.GetAttesterDuties(ctx, epoch, validatorIndices)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedAttesterDuties.Data, attesterDuties)
}

func TestGetAttesterDuties_HttpError(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		ctx,
		fmt.Sprintf("%s/%d", getAttesterDutiesTestEndpoint, epoch),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetAttesterDuties(ctx, epoch, nil)
	assert.ErrorContains(t, "foo error", err)
	assert.ErrorContains(t, "failed to send POST data to REST endpoint", err)
}

func TestGetAttesterDuties_NilAttesterDuty(t *testing.T) {
	const epoch = types.Epoch(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		ctx,
		fmt.Sprintf("%s/%d", getAttesterDutiesTestEndpoint, epoch),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		nil,
	).SetArg(
		4,
		apimiddleware.AttesterDutiesResponseJson{
			Data: []*apimiddleware.AttesterDutyJson{nil},
		},
	).Times(1)

	dutiesProvider := &beaconApiDutiesProvider{jsonRestHandler: jsonRestHandler}
	_, err := dutiesProvider.GetAttesterDuties(ctx, epoch, nil)
	assert.ErrorContains(t, "attester duty at index `0` is nil", err)
}
