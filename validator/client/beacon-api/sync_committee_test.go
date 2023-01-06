package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestSubmitSyncMessage_Valid(t *testing.T) {
	const beaconBlockRoot = "0x719d4f66a5f25c35d93718821aacb342194391034b11cf0a5822cc249178a274"
	const signature = "0xb459ef852bd4e0cb96e6723d67cacc8215406dd9ba663f8874a083167ebf428b28b746431bdbc1820a25289377b2610881e52b3a05c3548c5e99c08c8a36342573be5962d7510c03dcba8ddfb8ae419e59d222ddcf31cc512e704ef2cc3cf8"

	decodedBeaconBlockRoot, err := hexutil.Decode(beaconBlockRoot)
	require.NoError(t, err)

	decodedSignature, err := hexutil.Decode(signature)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonSyncCommitteeMessage := &apimiddleware.SyncCommitteeMessageJson{
		Slot:            "42",
		BeaconBlockRoot: beaconBlockRoot,
		ValidatorIndex:  "12345",
		Signature:       signature,
	}

	marshalledJsonRegistrations, err := json.Marshal(jsonSyncCommitteeMessage)
	require.NoError(t, err)

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		context.Background(),
		"/eth/v1/beacon/pool/sync_committees",
		nil,
		bytes.NewBuffer(marshalledJsonRegistrations),
		nil,
	).Return(
		nil,
		nil,
	).Times(1)

	protoSyncCommiteeMessage := ethpb.SyncCommitteeMessage{
		Slot:           types.Slot(42),
		BlockRoot:      decodedBeaconBlockRoot,
		ValidatorIndex: types.ValidatorIndex(12345),
		Signature:      decodedSignature,
	}

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	res, err := validatorClient.SubmitSyncMessage(context.Background(), &protoSyncCommiteeMessage)

	assert.DeepEqual(t, new(empty.Empty), res)
	require.NoError(t, err)
}

func TestSubmitSyncMessage_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().PostRestJson(
		context.Background(),
		"/eth/v1/beacon/pool/sync_committees",
		nil,
		gomock.Any(),
		nil,
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	_, err := validatorClient.SubmitSyncMessage(context.Background(), &ethpb.SyncCommitteeMessage{})
	assert.ErrorContains(t, "failed to send POST data to `/eth/v1/beacon/pool/sync_committees` REST endpoint", err)
	assert.ErrorContains(t, "foo error", err)
}
