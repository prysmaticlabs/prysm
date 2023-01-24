package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestCheckDoppelGanger_Nominal(t *testing.T) {
	const stringPubKey1 = "0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526"
	const stringPubKey2 = "0x80002662ecb857da7a37ed468291cb248979eca5131db56c20843262f7909220c296e18f59af1726ef86ec15c08b8317"
	const stringPubKey3 = "0x80003a1c67216514e4ab257738e59ef38063edf43bc4a2ef9d38633bdde117384401684c6cf81aa04cf18890e75ab52c"
	const stringPubKey4 = "0x80007e05ba643a3e5be65d1595154023dc2cf009626f32ab1054c5225a6beb28b8be3d52a463ab45f698df884614c87d"
	const stringPubKey5 = "0x80006ab8cd402459b445b2f5f955c9bae550bc269717837a8cd68176ce42a21fd372b844d508711d6e0bb0efe65abfe5"
	const stringPubKey6 = "0x800077c436fc0c57bec2b91509519deadeed235f35f6377e7865e17ee86271120381a49c643829be12d232a4ba8360d2"

	pubKey1, err := hexutil.Decode(stringPubKey1)
	require.NoError(t, err)

	pubKey2, err := hexutil.Decode(stringPubKey2)
	require.NoError(t, err)

	pubKey3, err := hexutil.Decode(stringPubKey3)
	require.NoError(t, err)

	pubKey4, err := hexutil.Decode(stringPubKey4)
	require.NoError(t, err)

	pubKey5, err := hexutil.Decode(stringPubKey5)
	require.NoError(t, err)

	pubKey6, err := hexutil.Decode(stringPubKey6)
	require.NoError(t, err)

	testCases := []struct {
		name                        string
		doppelGangerInput           *ethpb.DoppelGangerRequest
		doppelGangerExpectedOutput  *ethpb.DoppelGangerResponse
		getSyncingOutput            *apimiddleware.SyncingResponseJson
		getForkOutput               *apimiddleware.StateForkResponseJson
		getHeadersOutput            *apimiddleware.BlockHeadersResponseJson
		getStateValidatorsInterface *struct {
			input  []string
			output *apimiddleware.StateValidatorsResponseJson
		}
		getLivelinessInterfaces []struct {
			inputUrl           string
			inputStringIndexes []string
			output             *apimiddleware.LivenessResponseJson
		}
	}{
		{
			name:              "nil input",
			doppelGangerInput: nil,
			doppelGangerExpectedOutput: &ethpb.DoppelGangerResponse{
				Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{},
			},
		},
		{
			name: "nil validator requests",
			doppelGangerInput: &ethpb.DoppelGangerRequest{
				ValidatorRequests: nil,
			},
			doppelGangerExpectedOutput: &ethpb.DoppelGangerResponse{
				Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{},
			},
		},
		{
			name: "empty validator requests",
			doppelGangerInput: &ethpb.DoppelGangerRequest{
				ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{},
			},
			doppelGangerExpectedOutput: &ethpb.DoppelGangerResponse{
				Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{},
			},
		},
		{
			name: "phase0",
			doppelGangerInput: &ethpb.DoppelGangerRequest{
				ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{
					{PublicKey: pubKey1},
					{PublicKey: pubKey2},
					{PublicKey: pubKey3},
					{PublicKey: pubKey4},
					{PublicKey: pubKey5},
					{PublicKey: pubKey6},
				},
			},
			doppelGangerExpectedOutput: &ethpb.DoppelGangerResponse{
				Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{
					{PublicKey: pubKey1, DuplicateExists: false},
					{PublicKey: pubKey2, DuplicateExists: false},
					{PublicKey: pubKey3, DuplicateExists: false},
					{PublicKey: pubKey4, DuplicateExists: false},
					{PublicKey: pubKey5, DuplicateExists: false},
					{PublicKey: pubKey6, DuplicateExists: false},
				},
			},
			getSyncingOutput: &apimiddleware.SyncingResponseJson{
				Data: &helpers.SyncDetailsJson{
					IsSyncing: false,
				},
			},
			getForkOutput: &apimiddleware.StateForkResponseJson{
				Data: &apimiddleware.ForkJson{
					PreviousVersion: "0x00000000",
					CurrentVersion:  "0x00000000",
					Epoch:           "42",
				},
			},
		},
		{
			name: "all validators are recent",
			doppelGangerInput: &ethpb.DoppelGangerRequest{
				ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{
					{PublicKey: pubKey1, Epoch: 2},
					{PublicKey: pubKey2, Epoch: 2},
					{PublicKey: pubKey3, Epoch: 2},
					{PublicKey: pubKey4, Epoch: 2},
					{PublicKey: pubKey5, Epoch: 2},
					{PublicKey: pubKey6, Epoch: 2},
				},
			},
			doppelGangerExpectedOutput: &ethpb.DoppelGangerResponse{
				Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{
					{PublicKey: pubKey1, DuplicateExists: false},
					{PublicKey: pubKey2, DuplicateExists: false},
					{PublicKey: pubKey3, DuplicateExists: false},
					{PublicKey: pubKey4, DuplicateExists: false},
					{PublicKey: pubKey5, DuplicateExists: false},
					{PublicKey: pubKey6, DuplicateExists: false},
				},
			},
			getSyncingOutput: &apimiddleware.SyncingResponseJson{
				Data: &helpers.SyncDetailsJson{
					IsSyncing: false,
				},
			},
			getForkOutput: &apimiddleware.StateForkResponseJson{
				Data: &apimiddleware.ForkJson{
					PreviousVersion: "0x01000000",
					CurrentVersion:  "0x02000000",
					Epoch:           "2",
				},
			},
			getHeadersOutput: &apimiddleware.BlockHeadersResponseJson{
				Data: []*apimiddleware.BlockHeaderContainerJson{
					{
						Header: &apimiddleware.BeaconBlockHeaderContainerJson{
							Message: &apimiddleware.BeaconBlockHeaderJson{
								Slot: "99",
							},
						},
					},
				},
			},
		},
		{
			name: "some validators are recent, some not, some duplicates",
			doppelGangerInput: &ethpb.DoppelGangerRequest{
				ValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{
					{PublicKey: pubKey1, Epoch: 99}, // recent
					{PublicKey: pubKey2, Epoch: 80}, // not recent - duplicate on previous epoch
					{PublicKey: pubKey3, Epoch: 80}, // not recent - duplicate on current epoch
					{PublicKey: pubKey4, Epoch: 80}, // not recent - duplicate on both previous and current epoch
					{PublicKey: pubKey5, Epoch: 80}, // non existing validator
					{PublicKey: pubKey6, Epoch: 80}, // not recent - not duplicate
				},
			},
			doppelGangerExpectedOutput: &ethpb.DoppelGangerResponse{
				Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{
					{PublicKey: pubKey1, DuplicateExists: false}, // recent
					{PublicKey: pubKey2, DuplicateExists: true},  // not recent - duplicate on previous epoch
					{PublicKey: pubKey3, DuplicateExists: true},  // not recent - duplicate on current epoch
					{PublicKey: pubKey4, DuplicateExists: true},  // not recent - duplicate on both previous and current epoch
					{PublicKey: pubKey5, DuplicateExists: false}, // non existing validator
					{PublicKey: pubKey6, DuplicateExists: false}, // not recent - not duplicate
				},
			},
			getSyncingOutput: &apimiddleware.SyncingResponseJson{
				Data: &helpers.SyncDetailsJson{
					IsSyncing: false,
				},
			},
			getForkOutput: &apimiddleware.StateForkResponseJson{
				Data: &apimiddleware.ForkJson{
					PreviousVersion: "0x01000000",
					CurrentVersion:  "0x02000000",
					Epoch:           "2",
				},
			},
			getHeadersOutput: &apimiddleware.BlockHeadersResponseJson{
				Data: []*apimiddleware.BlockHeaderContainerJson{
					{
						Header: &apimiddleware.BeaconBlockHeaderContainerJson{
							Message: &apimiddleware.BeaconBlockHeaderJson{
								Slot: "3201",
							},
						},
					},
				},
			},
			getStateValidatorsInterface: &struct {
				input  []string
				output *apimiddleware.StateValidatorsResponseJson
			}{
				input: []string{
					// no stringPubKey1 since recent
					stringPubKey2, // not recent - duplicate on previous epoch
					stringPubKey3, // not recent - duplicate on current epoch
					stringPubKey4, // not recent - duplicate on both previous and current epoch
					stringPubKey5, // non existing validator
					stringPubKey6, // not recent - not duplicate
				},
				output: &apimiddleware.StateValidatorsResponseJson{
					Data: []*apimiddleware.ValidatorContainerJson{
						// No "11111" since corresponding validator is recent
						{Index: "22222", Validator: &apimiddleware.ValidatorJson{PublicKey: stringPubKey2}}, // not recent - duplicate on previous epoch
						{Index: "33333", Validator: &apimiddleware.ValidatorJson{PublicKey: stringPubKey3}}, // not recent - duplicate on current epoch
						{Index: "44444", Validator: &apimiddleware.ValidatorJson{PublicKey: stringPubKey4}}, // not recent - duplicate on both previous and current epoch
						// No "55555" sicee corresponding validator does not exist
						{Index: "66666", Validator: &apimiddleware.ValidatorJson{PublicKey: stringPubKey6}}, // not recent - not duplicate
					},
				},
			},
			getLivelinessInterfaces: []struct {
				inputUrl           string
				inputStringIndexes []string
				output             *apimiddleware.LivenessResponseJson
			}{
				{
					inputUrl: "/eth/v1/validator/liveness/99", // previous epoch
					inputStringIndexes: []string{
						// No "11111" since corresponding validator is recent
						"22222", // not recent - duplicate on previous epoch
						"33333", // not recent - duplicate on current epoch
						"44444", // not recent - duplicate on both previous and current epoch
						// No "55555" since corresponding validator it does not exist
						"66666", // not recent - not duplicate
					},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{
							// No "11111" since corresponding validator is recent
							{Index: "22222", IsLive: true},  // not recent - duplicate on previous epoch
							{Index: "33333", IsLive: false}, // not recent - duplicate on current epoch
							{Index: "44444", IsLive: true},  // not recent - duplicate on both previous and current epoch
							// No "55555" since corresponding validator it does not exist
							{Index: "66666", IsLive: false}, // not recent - not duplicate
						},
					},
				},
				{
					inputUrl: "/eth/v1/validator/liveness/100", // current epoch
					inputStringIndexes: []string{
						// No "11111" since corresponding validator is recent
						"22222", // not recent - duplicate on previous epoch
						"33333", // not recent - duplicate on current epoch
						"44444", // not recent - duplicate on both previous and current epoch
						// No "55555" since corresponding validator it does not exist
						"66666", // not recent - not duplicate
					},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{
							// No "11111" since corresponding validator is recent
							{Index: "22222", IsLive: false}, // not recent - duplicate on previous epoch
							{Index: "33333", IsLive: true},  // not recent - duplicate on current epoch
							{Index: "44444", IsLive: true},  // not recent - duplicate on both previous and current epoch
							// No "55555" since corresponding validator it does not exist
							{Index: "66666", IsLive: false}, // not recent - not duplicate
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

			ctx := context.Background()

			if testCase.getSyncingOutput != nil {
				syncingResponseJson := apimiddleware.SyncingResponseJson{}

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					syncingEnpoint,
					&syncingResponseJson,
				).Return(
					nil,
					nil,
				).SetArg(
					2,
					*testCase.getSyncingOutput,
				).Times(1)
			}

			if testCase.getForkOutput != nil {
				stateForkResponseJson := apimiddleware.StateForkResponseJson{}

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					forkEndpoint,
					&stateForkResponseJson,
				).Return(
					nil,
					nil,
				).SetArg(
					2,
					*testCase.getForkOutput,
				).Times(1)
			}

			if testCase.getHeadersOutput != nil {
				blockHeadersResponseJson := apimiddleware.BlockHeadersResponseJson{}

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					headersEndpoint,
					&blockHeadersResponseJson,
				).Return(
					nil,
					nil,
				).SetArg(
					2,
					*testCase.getHeadersOutput,
				).Times(1)
			}

			if testCase.getLivelinessInterfaces != nil {
				for _, iface := range testCase.getLivelinessInterfaces {
					livenessResponseJson := apimiddleware.LivenessResponseJson{}

					marshalledIndexes, err := json.Marshal(iface.inputStringIndexes)
					require.NoError(t, err)

					jsonRestHandler.EXPECT().PostRestJson(
						ctx,
						iface.inputUrl,
						nil,
						bytes.NewBuffer(marshalledIndexes),
						&livenessResponseJson,
					).SetArg(
						4,
						*iface.output,
					).Return(
						nil,
						nil,
					).Times(1)
				}
			}

			stateValidatorsProvider := mock.NewMockstateValidatorsProvider(ctrl)

			if testCase.getStateValidatorsInterface != nil {
				stateValidatorsProvider.EXPECT().GetStateValidators(
					ctx,
					testCase.getStateValidatorsInterface.input,
					nil,
					nil,
				).Return(
					testCase.getStateValidatorsInterface.output,
					nil,
				).Times(1)
			}

			validatorClient := beaconApiValidatorClient{
				jsonRestHandler:         jsonRestHandler,
				stateValidatorsProvider: stateValidatorsProvider,
			}

			doppelGangerActualOutput, err := validatorClient.CheckDoppelGanger(
				context.Background(),
				testCase.doppelGangerInput,
			)

			require.DeepEqual(t, testCase.doppelGangerExpectedOutput, doppelGangerActualOutput)
			assert.NoError(t, err)
		})
	}
}

func TestCheckDoppelGanger_Errors(t *testing.T) {
	const stringPubKey = "0x80000e851c0f53c3246ff726d7ff7766661ca5e12a07c45c114d208d54f0f8233d4380b2e9aff759d69795d1df905526"
	pubKey, err := hexutil.Decode(stringPubKey)
	require.NoError(t, err)

	standardInputValidatorRequests := []*ethpb.DoppelGangerRequest_ValidatorRequest{
		{
			PublicKey: pubKey,
			Epoch:     1,
		},
	}

	standardGetSyncingOutput := &apimiddleware.SyncingResponseJson{
		Data: &helpers.SyncDetailsJson{
			IsSyncing: false,
		},
	}

	standardGetForkOutput := &apimiddleware.StateForkResponseJson{
		Data: &apimiddleware.ForkJson{
			CurrentVersion: "0x02000000",
		},
	}

	standardGetHeadersOutput := &apimiddleware.BlockHeadersResponseJson{
		Data: []*apimiddleware.BlockHeaderContainerJson{
			{
				Header: &apimiddleware.BeaconBlockHeaderContainerJson{
					Message: &apimiddleware.BeaconBlockHeaderJson{
						Slot: "1000",
					},
				},
			},
		},
	}

	standardGetStateValidatorsInterface := &struct {
		input  []string
		output *apimiddleware.StateValidatorsResponseJson
		err    error
	}{
		input: []string{stringPubKey},
		output: &apimiddleware.StateValidatorsResponseJson{
			Data: []*apimiddleware.ValidatorContainerJson{
				{
					Index: "42",
					Validator: &apimiddleware.ValidatorJson{
						PublicKey: stringPubKey,
					},
				},
			},
		},
	}

	testCases := []struct {
		name                        string
		expectedErrorMessage        string
		inputValidatorRequests      []*ethpb.DoppelGangerRequest_ValidatorRequest
		getSyncingOutput            *apimiddleware.SyncingResponseJson
		getSyncingError             error
		getForkOutput               *apimiddleware.StateForkResponseJson
		getForkError                error
		getHeadersOutput            *apimiddleware.BlockHeadersResponseJson
		getHeadersError             error
		getStateValidatorsInterface *struct {
			input  []string
			output *apimiddleware.StateValidatorsResponseJson
			err    error
		}
		getLivenessInterfaces []struct {
			inputUrl           string
			inputStringIndexes []string
			output             *apimiddleware.LivenessResponseJson
			err                error
		}
	}{
		{
			name:                   "nil validatorRequest",
			expectedErrorMessage:   "validator request is nil",
			inputValidatorRequests: []*ethpb.DoppelGangerRequest_ValidatorRequest{nil},
		},
		{
			name:                   "isSyncing on error",
			expectedErrorMessage:   "failed to get beacon node sync status",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getSyncingError:        errors.New("custom error"),
		},
		{
			name:                   "beacon node not synced",
			expectedErrorMessage:   "beacon node not synced",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput: &apimiddleware.SyncingResponseJson{
				Data: &helpers.SyncDetailsJson{
					IsSyncing: true,
				},
			},
		},
		{
			name:                   "getFork on error",
			expectedErrorMessage:   "failed to get fork",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          &apimiddleware.StateForkResponseJson{},
			getForkError:           errors.New("custom error"),
		},
		{
			name:                   "cannot decode fork version",
			expectedErrorMessage:   "failed to decode fork version",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          &apimiddleware.StateForkResponseJson{Data: &apimiddleware.ForkJson{CurrentVersion: "not a version"}},
		},
		{
			name:                   "get headers on error",
			expectedErrorMessage:   "failed to get headers",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          standardGetForkOutput,
			getHeadersOutput:       &apimiddleware.BlockHeadersResponseJson{},
			getHeadersError:        errors.New("custom error"),
		},
		{
			name:                   "cannot parse head slot",
			expectedErrorMessage:   "failed to parse head slot",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          standardGetForkOutput,
			getHeadersOutput: &apimiddleware.BlockHeadersResponseJson{
				Data: []*apimiddleware.BlockHeaderContainerJson{
					{
						Header: &apimiddleware.BeaconBlockHeaderContainerJson{
							Message: &apimiddleware.BeaconBlockHeaderJson{
								Slot: "not a slot",
							},
						},
					},
				},
			},
		},
		{
			name:                   "state validators error",
			expectedErrorMessage:   "failed to get state validators",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          standardGetForkOutput,
			getHeadersOutput:       standardGetHeadersOutput,
			getStateValidatorsInterface: &struct {
				input  []string
				output *apimiddleware.StateValidatorsResponseJson
				err    error
			}{
				input: []string{stringPubKey},
				err:   errors.New("custom error"),
			},
		},
		{
			name:                   "validator container is nil",
			expectedErrorMessage:   "validator container is nil",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          standardGetForkOutput,
			getHeadersOutput:       standardGetHeadersOutput,
			getStateValidatorsInterface: &struct {
				input  []string
				output *apimiddleware.StateValidatorsResponseJson
				err    error
			}{
				input:  []string{stringPubKey},
				output: &apimiddleware.StateValidatorsResponseJson{Data: []*apimiddleware.ValidatorContainerJson{nil}},
			},
		},
		{
			name:                   "validator is nil",
			expectedErrorMessage:   "validator is nil",
			inputValidatorRequests: standardInputValidatorRequests,
			getSyncingOutput:       standardGetSyncingOutput,
			getForkOutput:          standardGetForkOutput,
			getHeadersOutput:       standardGetHeadersOutput,
			getStateValidatorsInterface: &struct {
				input  []string
				output *apimiddleware.StateValidatorsResponseJson
				err    error
			}{
				input:  []string{stringPubKey},
				output: &apimiddleware.StateValidatorsResponseJson{Data: []*apimiddleware.ValidatorContainerJson{{Validator: nil}}},
			},
		},
		{
			name:                        "previous epoch liveness error",
			expectedErrorMessage:        "failed to get map from validator index to liveness for previous epoch 30",
			inputValidatorRequests:      standardInputValidatorRequests,
			getSyncingOutput:            standardGetSyncingOutput,
			getForkOutput:               standardGetForkOutput,
			getHeadersOutput:            standardGetHeadersOutput,
			getStateValidatorsInterface: standardGetStateValidatorsInterface,
			getLivenessInterfaces: []struct {
				inputUrl           string
				inputStringIndexes []string
				output             *apimiddleware.LivenessResponseJson
				err                error
			}{
				{
					inputUrl:           "/eth/v1/validator/liveness/30",
					inputStringIndexes: []string{"42"},
					output:             &apimiddleware.LivenessResponseJson{},
					err:                errors.New("custom error"),
				},
			},
		},
		{
			name:                        "liveness is nil",
			expectedErrorMessage:        "liveness is nil",
			inputValidatorRequests:      standardInputValidatorRequests,
			getSyncingOutput:            standardGetSyncingOutput,
			getForkOutput:               standardGetForkOutput,
			getHeadersOutput:            standardGetHeadersOutput,
			getStateValidatorsInterface: standardGetStateValidatorsInterface,
			getLivenessInterfaces: []struct {
				inputUrl           string
				inputStringIndexes []string
				output             *apimiddleware.LivenessResponseJson
				err                error
			}{
				{
					inputUrl:           "/eth/v1/validator/liveness/30",
					inputStringIndexes: []string{"42"},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{nil},
					},
				},
			},
		},
		{
			name:                        "current epoch liveness error",
			expectedErrorMessage:        "failed to get map from validator index to liveness for current epoch 31",
			inputValidatorRequests:      standardInputValidatorRequests,
			getSyncingOutput:            standardGetSyncingOutput,
			getForkOutput:               standardGetForkOutput,
			getHeadersOutput:            standardGetHeadersOutput,
			getStateValidatorsInterface: standardGetStateValidatorsInterface,
			getLivenessInterfaces: []struct {
				inputUrl           string
				inputStringIndexes []string
				output             *apimiddleware.LivenessResponseJson
				err                error
			}{
				{
					inputUrl:           "/eth/v1/validator/liveness/30",
					inputStringIndexes: []string{"42"},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{},
					},
				},
				{
					inputUrl:           "/eth/v1/validator/liveness/31",
					inputStringIndexes: []string{"42"},
					output:             &apimiddleware.LivenessResponseJson{},
					err:                errors.New("custom error"),
				},
			},
		},
		{
			name:                        "wrong validator index for previous epoch",
			expectedErrorMessage:        "failed to retrieve liveness for previous epoch `30` for validator index `42`",
			inputValidatorRequests:      standardInputValidatorRequests,
			getSyncingOutput:            standardGetSyncingOutput,
			getForkOutput:               standardGetForkOutput,
			getHeadersOutput:            standardGetHeadersOutput,
			getStateValidatorsInterface: standardGetStateValidatorsInterface,
			getLivenessInterfaces: []struct {
				inputUrl           string
				inputStringIndexes []string
				output             *apimiddleware.LivenessResponseJson
				err                error
			}{
				{
					inputUrl:           "/eth/v1/validator/liveness/30",
					inputStringIndexes: []string{"42"},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{},
					},
				},
				{
					inputUrl:           "/eth/v1/validator/liveness/31",
					inputStringIndexes: []string{"42"},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{},
					},
				},
			},
		},
		{
			name:                        "wrong validator index for current epoch",
			expectedErrorMessage:        "failed to retrieve liveness for current epoch `31` for validator index `42`",
			inputValidatorRequests:      standardInputValidatorRequests,
			getSyncingOutput:            standardGetSyncingOutput,
			getForkOutput:               standardGetForkOutput,
			getHeadersOutput:            standardGetHeadersOutput,
			getStateValidatorsInterface: standardGetStateValidatorsInterface,
			getLivenessInterfaces: []struct {
				inputUrl           string
				inputStringIndexes []string
				output             *apimiddleware.LivenessResponseJson
				err                error
			}{
				{
					inputUrl:           "/eth/v1/validator/liveness/30",
					inputStringIndexes: []string{"42"},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{
							{
								Index: "42",
							},
						},
					},
				},
				{
					inputUrl:           "/eth/v1/validator/liveness/31",
					inputStringIndexes: []string{"42"},
					output: &apimiddleware.LivenessResponseJson{
						Data: []*struct {
							Index  string `json:"index"`
							IsLive bool   `json:"is_live"`
						}{},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

			ctx := context.Background()

			if testCase.getSyncingOutput != nil {
				syncingResponseJson := apimiddleware.SyncingResponseJson{}

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					syncingEnpoint,
					&syncingResponseJson,
				).Return(
					nil,
					testCase.getSyncingError,
				).SetArg(
					2,
					*testCase.getSyncingOutput,
				).Times(1)
			}

			if testCase.getForkOutput != nil {
				stateForkResponseJson := apimiddleware.StateForkResponseJson{}

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					forkEndpoint,
					&stateForkResponseJson,
				).Return(
					nil,
					testCase.getForkError,
				).SetArg(
					2,
					*testCase.getForkOutput,
				).Times(1)
			}

			if testCase.getHeadersOutput != nil {
				blockHeadersResponseJson := apimiddleware.BlockHeadersResponseJson{}

				jsonRestHandler.EXPECT().GetRestJsonResponse(
					ctx,
					headersEndpoint,
					&blockHeadersResponseJson,
				).Return(
					nil,
					testCase.getHeadersError,
				).SetArg(
					2,
					*testCase.getHeadersOutput,
				).Times(1)
			}

			stateValidatorsProvider := mock.NewMockstateValidatorsProvider(ctrl)

			if testCase.getStateValidatorsInterface != nil {
				stateValidatorsProvider.EXPECT().GetStateValidators(
					ctx,
					testCase.getStateValidatorsInterface.input,
					nil,
					nil,
				).Return(
					testCase.getStateValidatorsInterface.output,
					testCase.getStateValidatorsInterface.err,
				).Times(1)
			}

			if testCase.getLivenessInterfaces != nil {
				for _, iface := range testCase.getLivenessInterfaces {
					livenessResponseJson := apimiddleware.LivenessResponseJson{}

					marshalledIndexes, err := json.Marshal(iface.inputStringIndexes)
					require.NoError(t, err)

					jsonRestHandler.EXPECT().PostRestJson(
						ctx,
						iface.inputUrl,
						nil,
						bytes.NewBuffer(marshalledIndexes),
						&livenessResponseJson,
					).SetArg(
						4,
						*iface.output,
					).Return(
						nil,
						iface.err,
					).Times(1)
				}
			}

			validatorClient := beaconApiValidatorClient{
				jsonRestHandler:         jsonRestHandler,
				stateValidatorsProvider: stateValidatorsProvider,
			}

			_, err := validatorClient.CheckDoppelGanger(
				context.Background(),
				&ethpb.DoppelGangerRequest{
					ValidatorRequests: testCase.inputValidatorRequests,
				},
			)

			require.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}
