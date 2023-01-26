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

func TestGetBeaconBlock_CapellaValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	capellaProtoBeaconBlock := test_helpers.GenerateProtoCapellaBeaconBlock()
	capellaBeaconBlockBytes, err := json.Marshal(test_helpers.GenerateJsonCapellaBeaconBlock())
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
			Version: "capella",
			Data:    capellaBeaconBlockBytes,
		},
	).Return(
		nil,
		nil,
	).Times(1)

	expectedBeaconBlock := generateProtoCapellaBlock(capellaProtoBeaconBlock)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_CapellaError(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *apimiddleware.BeaconBlockCapellaJson
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *apimiddleware.BeaconBlockCapellaJson {
				beaconBlock := test_helpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil execution payload",
			expectedErrorMessage: "execution payload is nil",
			generateData: func() *apimiddleware.BeaconBlockCapellaJson {
				beaconBlock := test_helpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.ExecutionPayload = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad bellatrix fields",
			expectedErrorMessage: "failed to get the bellatrix fields of the capella block",
			generateData: func() *apimiddleware.BeaconBlockCapellaJson {
				beaconBlock := test_helpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad withdrawals",
			expectedErrorMessage: "failed to get withdrawals",
			generateData: func() *apimiddleware.BeaconBlockCapellaJson {
				beaconBlock := test_helpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.ExecutionPayload.Withdrawals[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad bls execution changes",
			expectedErrorMessage: "failed to get bls to execution changes",
			generateData: func() *apimiddleware.BeaconBlockCapellaJson {
				beaconBlock := test_helpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.BLSToExecutionChanges[0] = nil
				return beaconBlock
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			dataBytes, err := json.Marshal(testCase.generateData())
			require.NoError(t, err)

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				context.Background(),
				gomock.Any(),
				&abstractProduceBlockResponseJson{},
			).SetArg(
				2,
				abstractProduceBlockResponseJson{
					Version: "capella",
					Data:    dataBytes,
				},
			).Return(
				nil,
				nil,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			_, err = validatorClient.getBeaconBlock(context.Background(), 1, []byte{1}, []byte{2})
			assert.ErrorContains(t, "failed to get capella block", err)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func generateProtoCapellaBlock(capellaProtoBeaconBlock *ethpb.BeaconBlockCapella) *ethpb.GenericBeaconBlock {
	return &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Capella{
			Capella: capellaProtoBeaconBlock,
		},
	}
}
