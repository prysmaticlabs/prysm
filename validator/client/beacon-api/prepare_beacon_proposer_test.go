package beacon_api

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestPrepareBeaconProposer_Valid(t *testing.T) {
	const feeRecipient1 = "0x64d0f08d61388b2c7eb8ef4eb69e649daaf492890225cb525122d02f2da52a4d"
	const feeRecipient2 = "0x69b166126aced90c8ed9a78ceba667d2232fab95ed5bbd104a4bf9e46ae0de76"
	const feeRecipient3 = "0x00438a2bb632a91f335de330dd84b75cf28c5546c8335d9d18137402ea1f25ef"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRecipients := []*apimiddleware.FeeRecipientJson{
		{
			ValidatorIndex: "1",
			FeeRecipient:   feeRecipient1,
		},
		{
			ValidatorIndex: "2",
			FeeRecipient:   feeRecipient2,
		},
		{
			ValidatorIndex: "3",
			FeeRecipient:   feeRecipient3,
		},
	}

	marshalledJsonRecipients, err := json.Marshal(jsonRecipients)
	require.NoError(t, err)

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		"/eth/v1/validator/prepare_beacon_proposer",
		nil,
		bytes.NewBuffer(marshalledJsonRecipients),
		nil,
	).Return(
		nil,
		nil,
	).Times(1)

	decodedFeeRecipient1, err := hexutil.Decode(feeRecipient1)
	require.NoError(t, err)
	decodedFeeRecipient2, err := hexutil.Decode(feeRecipient2)
	require.NoError(t, err)
	decodedFeeRecipient3, err := hexutil.Decode(feeRecipient3)
	require.NoError(t, err)

	protoRecipients := []*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer{
		{
			ValidatorIndex: 1,
			FeeRecipient:   decodedFeeRecipient1,
		},
		{
			ValidatorIndex: 2,
			FeeRecipient:   decodedFeeRecipient2,
		},
		{
			ValidatorIndex: 3,
			FeeRecipient:   decodedFeeRecipient3,
		},
	}

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	err = validatorClient.prepareBeaconProposer(protoRecipients)
	require.NoError(t, err)
}

func TestPrepareBeaconProposer_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		"/eth/v1/validator/prepare_beacon_proposer",
		nil,
		gomock.Any(),
		nil,
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	err := validatorClient.prepareBeaconProposer(nil)
	assert.ErrorContains(t, "failed to send POST data to REST endpoint", err)
	assert.ErrorContains(t, "foo error", err)
}
