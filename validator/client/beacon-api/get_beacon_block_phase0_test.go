package beacon_api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/test-helpers"
)

func TestGetBeaconBlock_Phase0Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	phase0ProtoBeaconBlock := test_helpers.GenerateProtoPhase0BeaconBlock()
	phase0BeaconBlockBytes, err := json.Marshal(test_helpers.GenerateJsonPhase0BeaconBlock())
	require.NoError(t, err)

	const slot = types.Slot(1)
	randaoReveal := []byte{2}
	graffiti := []byte{3}

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		fmt.Sprintf("/eth/v2/validator/blocks/%d?graffiti=%s&randao_reveal=%s", slot, hexutil.Encode(graffiti), hexutil.Encode(randaoReveal)),
		&abstractProduceBlockResponseJson{},
	).SetArg(
		1,
		abstractProduceBlockResponseJson{
			Version: "phase0",
			Data:    phase0BeaconBlockBytes,
		},
	).Return(
		nil,
		nil,
	).Times(1)

	expectedBeaconBlock := generateProtoPhase0Block(phase0ProtoBeaconBlock)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	beaconBlock, err := validatorClient.getBeaconBlock(slot, randaoReveal, graffiti)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_Phase0Error(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *apimiddleware.BeaconBlockJson
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil eth1 data",
			expectedErrorMessage: "eth1 data is nil",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad slot",
			expectedErrorMessage: "failed to parse slot `foo`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Slot = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad proposer index",
			expectedErrorMessage: "failed to parse proposer index `bar`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.ProposerIndex = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad parent root",
			expectedErrorMessage: "failed to decode parent root `foo`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.ParentRoot = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad state root",
			expectedErrorMessage: "failed to decode state root `bar`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.StateRoot = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad randao reveal",
			expectedErrorMessage: "failed to decode randao reveal `foo`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.RandaoReveal = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad deposit root",
			expectedErrorMessage: "failed to decode deposit root `bar`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data.DepositRoot = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad deposit count",
			expectedErrorMessage: "failed to parse deposit count `foo`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data.DepositCount = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad block hash",
			expectedErrorMessage: "failed to decode block hash `bar`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data.BlockHash = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad graffiti",
			expectedErrorMessage: "failed to decode graffiti `foo`",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Graffiti = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad proposer slashings",
			expectedErrorMessage: "failed to get proposer slashings",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.ProposerSlashings[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad attester slashings",
			expectedErrorMessage: "failed to get attester slashings",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.AttesterSlashings[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad attestations",
			expectedErrorMessage: "failed to get attestations",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Attestations[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad deposits",
			expectedErrorMessage: "failed to get deposits",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Deposits[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad voluntary exits",
			expectedErrorMessage: "failed to get voluntary exits",
			generateData: func() *apimiddleware.BeaconBlockJson {
				beaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.VoluntaryExits[0] = nil
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
				gomock.Any(),
				&abstractProduceBlockResponseJson{},
			).SetArg(
				1,
				abstractProduceBlockResponseJson{
					Version: "phase0",
					Data:    dataBytes,
				},
			).Return(
				nil,
				nil,
			).Times(1)

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
			_, err = validatorClient.getBeaconBlock(1, []byte{1}, []byte{2})
			assert.ErrorContains(t, "failed to get phase0 block", err)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func generateProtoPhase0Block(phase0ProtoBeaconBlock *ethpb.BeaconBlock) *ethpb.GenericBeaconBlock {
	return &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Phase0{
			Phase0: phase0ProtoBeaconBlock,
		},
	}
}
