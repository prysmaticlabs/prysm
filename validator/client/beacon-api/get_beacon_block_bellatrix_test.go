package beacon_api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/test-helpers"
)

func TestGetBeaconBlock_BellatrixValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bellatrixProtoBeaconBlock := test_helpers.GenerateProtoBellatrixBeaconBlock()
	bellatrixBeaconBlockBytes, err := json.Marshal(test_helpers.GenerateJsonBellatrixBeaconBlock())
	require.NoError(t, err)

	const slot = primitives.Slot(1)
	randaoReveal := []byte{2}
	graffiti := []byte{3}

	ctx := context.Background()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		fmt.Sprintf("/eth/v2/validator/blocks/%d?graffiti=%s&randao_reveal=%s", slot, hexutil.Encode(graffiti), hexutil.Encode(randaoReveal)),
		&abstractProduceBlockResponseJson{},
	).SetArg(
		2,
		abstractProduceBlockResponseJson{
			Version: "bellatrix",
			Data:    bellatrixBeaconBlockBytes,
		},
	).Return(
		nil,
		nil,
	).Times(1)

	expectedBeaconBlock := generateProtoBellatrixBlock(bellatrixProtoBeaconBlock)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_BellatrixError(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *apimiddleware.BeaconBlockBellatrixJson
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil execution payload",
			expectedErrorMessage: "execution payload is nil",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad altair fields",
			expectedErrorMessage: "failed to get the altair fields of the bellatrix block",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad parent hash",
			expectedErrorMessage: "failed to decode execution payload parent hash `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.ParentHash = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad fee recipient",
			expectedErrorMessage: "failed to decode execution payload fee recipient `bar`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.FeeRecipient = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad state root",
			expectedErrorMessage: "failed to decode execution payload state root `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.StateRoot = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad receipts root",
			expectedErrorMessage: "failed to decode execution payload receipts root `bar`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.ReceiptsRoot = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad logs bloom",
			expectedErrorMessage: "failed to decode execution payload logs bloom `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.LogsBloom = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad prev randao",
			expectedErrorMessage: "failed to decode execution payload prev randao `bar`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.PrevRandao = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad block number",
			expectedErrorMessage: "failed to parse execution payload block number `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.BlockNumber = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad gas limit",
			expectedErrorMessage: "failed to parse execution payload gas limit `bar`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.GasLimit = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad gas used",
			expectedErrorMessage: "failed to parse execution payload gas used `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.GasUsed = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad timestamp",
			expectedErrorMessage: "failed to parse execution payload timestamp `bar`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.TimeStamp = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad extra data",
			expectedErrorMessage: "failed to decode execution payload extra data `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.ExtraData = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad base fee per gas",
			expectedErrorMessage: "failed to parse execution payload base fee per gas `bar`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.BaseFeePerGas = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad block hash",
			expectedErrorMessage: "failed to decode execution payload block hash `foo`",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.BlockHash = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad transactions",
			expectedErrorMessage: "failed to get execution payload transactions",
			generateData: func() *apimiddleware.BeaconBlockBellatrixJson {
				beaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.Transactions[0] = "bar"
				return beaconBlock
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			dataBytes, err := json.Marshal(testCase.generateData())
			require.NoError(t, err)

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				gomock.Any(),
				&abstractProduceBlockResponseJson{},
			).SetArg(
				2,
				abstractProduceBlockResponseJson{
					Version: "bellatrix",
					Data:    dataBytes,
				},
			).Return(
				nil,
				nil,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			_, err = validatorClient.getBeaconBlock(ctx, 1, []byte{1}, []byte{2})
			assert.ErrorContains(t, "failed to get bellatrix block", err)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func generateProtoBellatrixBlock(bellatrixProtoBeaconBlock *ethpb.BeaconBlockBellatrix) *ethpb.GenericBeaconBlock {
	return &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Bellatrix{
			Bellatrix: bellatrixProtoBeaconBlock,
		},
	}
}
