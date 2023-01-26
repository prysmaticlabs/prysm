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

func TestGetBeaconBlock_AltairValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	altairProtoBeaconBlock := test_helpers.GenerateProtoAltairBeaconBlock()
	altairBeaconBlockBytes, err := json.Marshal(test_helpers.GenerateJsonAltairBeaconBlock())
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
			Version: "altair",
			Data:    altairBeaconBlockBytes,
		},
	).Return(
		nil,
		nil,
	).Times(1)

	expectedBeaconBlock := generateProtoAltairBlock(altairProtoBeaconBlock)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_AltairError(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *apimiddleware.BeaconBlockAltairJson
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *apimiddleware.BeaconBlockAltairJson {
				beaconBlock := test_helpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil sync aggregate",
			expectedErrorMessage: "sync aggregate is nil",
			generateData: func() *apimiddleware.BeaconBlockAltairJson {
				beaconBlock := test_helpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body.SyncAggregate = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad phase0 fields",
			expectedErrorMessage: "failed to get the phase0 fields of the altair block",
			generateData: func() *apimiddleware.BeaconBlockAltairJson {
				beaconBlock := test_helpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad sync committee bits",
			expectedErrorMessage: "failed to decode sync committee bits `foo`",
			generateData: func() *apimiddleware.BeaconBlockAltairJson {
				beaconBlock := test_helpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body.SyncAggregate.SyncCommitteeBits = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad sync committee signature",
			expectedErrorMessage: "failed to decode sync committee signature `bar`",
			generateData: func() *apimiddleware.BeaconBlockAltairJson {
				beaconBlock := test_helpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body.SyncAggregate.SyncCommitteeSignature = "bar"
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

			ctx := context.Background()

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				gomock.Any(),
				&abstractProduceBlockResponseJson{},
			).SetArg(
				2,
				abstractProduceBlockResponseJson{
					Version: "altair",
					Data:    dataBytes,
				},
			).Return(
				nil,
				nil,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			_, err = validatorClient.getBeaconBlock(ctx, 1, []byte{1}, []byte{2})
			assert.ErrorContains(t, "failed to get altair block", err)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func generateProtoAltairBlock(altairProtoBeaconBlock *ethpb.BeaconBlockAltair) *ethpb.GenericBeaconBlock {
	return &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Altair{
			Altair: altairProtoBeaconBlock,
		},
	}
}
