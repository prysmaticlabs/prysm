package beacon_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	rpctesting "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared/testing"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/test-helpers"
)

func TestGetBeaconBlock_RequestFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		gomock.Any(),
		gomock.Any(),
	).Return(
		nil,
		errors.New("foo error"),
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	_, err := validatorClient.getBeaconBlock(ctx, 1, []byte{1}, []byte{2})
	assert.ErrorContains(t, "failed to query GET REST endpoint", err)
	assert.ErrorContains(t, "foo error", err)
}

func TestGetBeaconBlock_Error(t *testing.T) {
	phase0BeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockJson{})
	require.NoError(t, err)
	altairBeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockAltairJson{})
	require.NoError(t, err)
	bellatrixBeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockBellatrixJson{})
	require.NoError(t, err)
	capellaBeaconBlockBytes, err := json.Marshal(apimiddleware.BeaconBlockCapellaJson{})
	require.NoError(t, err)

	testCases := []struct {
		name                 string
		beaconBlock          interface{}
		expectedErrorMessage string
		consensusVersion     string
		data                 json.RawMessage
	}{
		{
			name:                 "phase0 block decoding failed",
			expectedErrorMessage: "failed to decode phase0 block response json",
			consensusVersion:     "phase0",
			data:                 []byte{},
		},
		{
			name:                 "phase0 block conversion failed",
			expectedErrorMessage: "failed to get phase0 block",
			consensusVersion:     "phase0",
			data:                 phase0BeaconBlockBytes,
		},
		{
			name:                 "altair block decoding failed",
			expectedErrorMessage: "failed to decode altair block response json",
			consensusVersion:     "altair",
			data:                 []byte{},
		},
		{
			name:                 "altair block conversion failed",
			expectedErrorMessage: "failed to get altair block",
			consensusVersion:     "altair",
			data:                 altairBeaconBlockBytes,
		},
		{
			name:                 "bellatrix block decoding failed",
			expectedErrorMessage: "failed to decode bellatrix block response json",
			beaconBlock:          "foo",
			consensusVersion:     "bellatrix",
			data:                 []byte{},
		},
		{
			name:                 "bellatrix block conversion failed",
			expectedErrorMessage: "failed to get bellatrix block",
			consensusVersion:     "bellatrix",
			data:                 bellatrixBeaconBlockBytes,
		},
		{
			name:                 "capella block decoding failed",
			expectedErrorMessage: "failed to decode capella block response json",
			beaconBlock:          "foo",
			consensusVersion:     "capella",
			data:                 []byte{},
		},
		{
			name:                 "capella block conversion failed",
			expectedErrorMessage: "failed to get capella block",
			consensusVersion:     "capella",
			data:                 capellaBeaconBlockBytes,
		},
		{
			name:                 "unsupported consensus version",
			expectedErrorMessage: "unsupported consensus version `foo`",
			consensusVersion:     "foo",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
			jsonRestHandler.EXPECT().GetRestJsonResponse(
				ctx,
				gomock.Any(),
				&abstractProduceBlockResponseJson{},
			).SetArg(
				2,
				abstractProduceBlockResponseJson{
					Version: testCase.consensusVersion,
					Data:    testCase.data,
				},
			).Return(
				nil,
				nil,
			).Times(1)

			beaconBlockConverter := mock.NewMockbeaconBlockConverter(ctrl)
			beaconBlockConverter.EXPECT().ConvertRESTPhase0BlockToProto(
				gomock.Any(),
			).Return(
				nil,
				errors.New(testCase.expectedErrorMessage),
			).AnyTimes()

			beaconBlockConverter.EXPECT().ConvertRESTAltairBlockToProto(
				gomock.Any(),
			).Return(
				nil,
				errors.New(testCase.expectedErrorMessage),
			).AnyTimes()

			beaconBlockConverter.EXPECT().ConvertRESTBellatrixBlockToProto(
				gomock.Any(),
			).Return(
				nil,
				errors.New(testCase.expectedErrorMessage),
			).AnyTimes()

			beaconBlockConverter.EXPECT().ConvertRESTCapellaBlockToProto(
				gomock.Any(),
			).Return(
				nil,
				errors.New(testCase.expectedErrorMessage),
			).AnyTimes()

			validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler, beaconBlockConverter: beaconBlockConverter}
			_, err := validatorClient.getBeaconBlock(ctx, 1, []byte{1}, []byte{2})
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func TestGetBeaconBlock_Phase0Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	phase0ProtoBeaconBlock := test_helpers.GenerateProtoPhase0BeaconBlock()
	phase0BeaconBlock := test_helpers.GenerateJsonPhase0BeaconBlock()
	phase0BeaconBlockBytes, err := json.Marshal(phase0BeaconBlock)
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
			Version: "phase0",
			Data:    phase0BeaconBlockBytes,
		},
	).Return(
		nil,
		nil,
	).Times(1)

	beaconBlockConverter := mock.NewMockbeaconBlockConverter(ctrl)
	beaconBlockConverter.EXPECT().ConvertRESTPhase0BlockToProto(
		phase0BeaconBlock,
	).Return(
		phase0ProtoBeaconBlock,
		nil,
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler, beaconBlockConverter: beaconBlockConverter}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)

	expectedBeaconBlock := &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Phase0{
			Phase0: phase0ProtoBeaconBlock,
		},
	}

	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_AltairValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	altairProtoBeaconBlock := test_helpers.GenerateProtoAltairBeaconBlock()
	altairBeaconBlock := test_helpers.GenerateJsonAltairBeaconBlock()
	altairBeaconBlockBytes, err := json.Marshal(altairBeaconBlock)
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

	beaconBlockConverter := mock.NewMockbeaconBlockConverter(ctrl)
	beaconBlockConverter.EXPECT().ConvertRESTAltairBlockToProto(
		altairBeaconBlock,
	).Return(
		altairProtoBeaconBlock,
		nil,
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler, beaconBlockConverter: beaconBlockConverter}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)

	expectedBeaconBlock := &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Altair{
			Altair: altairProtoBeaconBlock,
		},
	}

	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_BellatrixValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bellatrixProtoBeaconBlock := test_helpers.GenerateProtoBellatrixBeaconBlock()
	bellatrixBeaconBlock := test_helpers.GenerateJsonBellatrixBeaconBlock()
	bellatrixBeaconBlockBytes, err := json.Marshal(bellatrixBeaconBlock)
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

	beaconBlockConverter := mock.NewMockbeaconBlockConverter(ctrl)
	beaconBlockConverter.EXPECT().ConvertRESTBellatrixBlockToProto(
		bellatrixBeaconBlock,
	).Return(
		bellatrixProtoBeaconBlock,
		nil,
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler, beaconBlockConverter: beaconBlockConverter}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)

	expectedBeaconBlock := &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Bellatrix{
			Bellatrix: bellatrixProtoBeaconBlock,
		},
	}

	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_CapellaValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	capellaProtoBeaconBlock := test_helpers.GenerateProtoCapellaBeaconBlock()
	capellaBeaconBlock := test_helpers.GenerateJsonCapellaBeaconBlock()
	capellaBeaconBlockBytes, err := json.Marshal(capellaBeaconBlock)
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

	beaconBlockConverter := mock.NewMockbeaconBlockConverter(ctrl)
	beaconBlockConverter.EXPECT().ConvertRESTCapellaBlockToProto(
		capellaBeaconBlock,
	).Return(
		capellaProtoBeaconBlock,
		nil,
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler, beaconBlockConverter: beaconBlockConverter}
	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)

	expectedBeaconBlock := &ethpb.GenericBeaconBlock{
		Block: &ethpb.GenericBeaconBlock_Capella{
			Capella: capellaProtoBeaconBlock,
		},
	}

	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlock_DenebValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var blockContents shared.SignedBeaconBlockContentsDeneb
	err := json.Unmarshal([]byte(rpctesting.DenebBlockContents), &blockContents)
	require.NoError(t, err)

	denebBeaconBlockBytes, err := json.Marshal(blockContents.ToUnsigned())
	require.NoError(t, err)
	ctx := context.Background()
	const slot = primitives.Slot(1)
	randaoReveal, err := hexutil.Decode("0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505")
	require.NoError(t, err)
	graffiti, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)

	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)
	jsonRestHandler.EXPECT().GetRestJsonResponse(
		ctx,
		fmt.Sprintf("/eth/v2/validator/blocks/%d?graffiti=%s&randao_reveal=%s", slot, hexutil.Encode(graffiti), hexutil.Encode(randaoReveal)),
		&abstractProduceBlockResponseJson{},
	).SetArg(
		2,
		abstractProduceBlockResponseJson{
			Version: "deneb",
			Data:    denebBeaconBlockBytes,
		},
	).Return(
		nil,
		nil,
	).Times(1)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}

	beaconBlock, err := validatorClient.getBeaconBlock(ctx, slot, randaoReveal, graffiti)
	require.NoError(t, err)

	expectedBeaconBlock, err := blockContents.ToUnsigned().ToGeneric()
	require.NoError(t, err)

	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}
