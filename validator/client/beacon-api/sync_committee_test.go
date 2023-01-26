package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
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

	marshalledJsonRegistrations, err := json.Marshal([]*apimiddleware.SyncCommitteeMessageJson{jsonSyncCommitteeMessage})
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
		Slot:           primitives.Slot(42),
		BlockRoot:      decodedBeaconBlockRoot,
		ValidatorIndex: primitives.ValidatorIndex(12345),
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

func TestGetSyncMessageBlockRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const blockRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	tests := []struct {
		name                 string
		endpointError        error
		expectedErrorMessage string
		expectedResponse     apimiddleware.BlockRootResponseJson
	}{
		{
			name: "valid request",
			expectedResponse: apimiddleware.BlockRootResponseJson{
				Data: &apimiddleware.BlockRootContainerJson{
					Root: blockRoot,
				},
			},
		},
		{
			name:                 "internal server error",
			expectedErrorMessage: "internal server error",
			endpointError:        errors.New("internal server error"),
		},
		{
			name: "execution optimistic",
			expectedResponse: apimiddleware.BlockRootResponseJson{
				ExecutionOptimistic: true,
			},
			expectedErrorMessage: "the node is currently optimistic and cannot serve validators",
		},
		{
			name:                 "no data",
			expectedResponse:     apimiddleware.BlockRootResponseJson{},
			expectedErrorMessage: "no data returned",
		},
		{
			name: "no root",
			expectedResponse: apimiddleware.BlockRootResponseJson{
				Data: new(apimiddleware.BlockRootContainerJson),
			},
			expectedErrorMessage: "no root returned",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				"/eth/v1/beacon/blocks/head/root",
				&apimiddleware.BlockRootResponseJson{},
			).SetArg(
				2,
				test.expectedResponse,
			).Return(
				nil,
				test.endpointError,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			actualResponse, err := validatorClient.getSyncMessageBlockRoot(ctx)
			if test.expectedErrorMessage != "" {
				require.ErrorContains(t, test.expectedErrorMessage, err)
				return
			}

			require.NoError(t, err)

			expectedRootBytes, err := hexutil.Decode(test.expectedResponse.Data.Root)
			require.NoError(t, err)

			require.NoError(t, err)
			require.DeepEqual(t, expectedRootBytes, actualResponse.Root)
		})
	}
}

func TestGetSyncCommitteeContribution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const blockRoot = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"

	request := &ethpb.SyncCommitteeContributionRequest{
		Slot:      primitives.Slot(1),
		PublicKey: nil,
		SubnetId:  1,
	}

	contributionJson := &apimiddleware.SyncCommitteeContributionJson{
		Slot:              "1",
		BeaconBlockRoot:   blockRoot,
		SubcommitteeIndex: "1",
		AggregationBits:   "0x01",
		Signature:         "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
	}

	tests := []struct {
		name           string
		contribution   apimiddleware.ProduceSyncCommitteeContributionResponseJson
		endpointErr    error
		expectedErrMsg string
	}{
		{
			name:         "valid request",
			contribution: apimiddleware.ProduceSyncCommitteeContributionResponseJson{Data: contributionJson},
		},
		{
			name:           "bad request",
			endpointErr:    errors.New("internal server error"),
			expectedErrMsg: "internal server error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				"/eth/v1/beacon/blocks/head/root",
				&apimiddleware.BlockRootResponseJson{},
			).SetArg(
				2,
				apimiddleware.BlockRootResponseJson{
					Data: &apimiddleware.BlockRootContainerJson{
						Root: blockRoot,
					},
				},
			).Return(
				nil,
				nil,
			).Times(1)

			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				fmt.Sprintf("/eth/v1/validator/sync_committee_contribution?slot=%d&subcommittee_index=%d&beacon_block_root=%s",
					uint64(request.Slot), request.SubnetId, blockRoot),
				&apimiddleware.ProduceSyncCommitteeContributionResponseJson{},
			).SetArg(
				2,
				test.contribution,
			).Return(
				nil,
				test.endpointErr,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			actualResponse, err := validatorClient.getSyncCommitteeContribution(ctx, request)
			if test.expectedErrMsg != "" {
				require.ErrorContains(t, test.expectedErrMsg, err)
				return
			}
			require.NoError(t, err)

			expectedResponse, err := convertSyncContributionJsonToProto(test.contribution.Data)
			require.NoError(t, err)
			assert.DeepEqual(t, expectedResponse, actualResponse)
		})
	}
}
