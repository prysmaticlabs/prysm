package beacon_api

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/test-helpers"
)

func TestProposeBeaconBlock_Phase0(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	phase0Block := generateSignedPhase0Block()

	genericSignedBlock := &ethpb.GenericSignedBeaconBlock{}
	genericSignedBlock.Block = phase0Block

	jsonPhase0Block := &apimiddleware.SignedBeaconBlockContainerJson{
		Signature: hexutil.Encode(phase0Block.Phase0.Signature),
		Message: &apimiddleware.BeaconBlockJson{
			ParentRoot:    hexutil.Encode(phase0Block.Phase0.Block.ParentRoot),
			ProposerIndex: uint64ToString(phase0Block.Phase0.Block.ProposerIndex),
			Slot:          uint64ToString(phase0Block.Phase0.Block.Slot),
			StateRoot:     hexutil.Encode(phase0Block.Phase0.Block.StateRoot),
			Body: &apimiddleware.BeaconBlockBodyJson{
				Attestations:      jsonifyAttestations(phase0Block.Phase0.Block.Body.Attestations),
				AttesterSlashings: jsonifyAttesterSlashings(phase0Block.Phase0.Block.Body.AttesterSlashings),
				Deposits:          jsonifyDeposits(phase0Block.Phase0.Block.Body.Deposits),
				Eth1Data:          jsonifyEth1Data(phase0Block.Phase0.Block.Body.Eth1Data),
				Graffiti:          hexutil.Encode(phase0Block.Phase0.Block.Body.Graffiti),
				ProposerSlashings: jsonifyProposerSlashings(phase0Block.Phase0.Block.Body.ProposerSlashings),
				RandaoReveal:      hexutil.Encode(phase0Block.Phase0.Block.Body.RandaoReveal),
				VoluntaryExits:    jsonifySignedVoluntaryExits(phase0Block.Phase0.Block.Body.VoluntaryExits),
			},
		},
	}

	marshalledBlock, err := json.Marshal(jsonPhase0Block)
	require.NoError(t, err)

	// Make sure that what we send in the POST body is the marshalled version of the protobuf block
	headers := map[string]string{"Eth-Consensus-Version": "phase0"}
	jsonRestHandler.EXPECT().PostRestJson(
		"/eth/v1/beacon/blocks",
		headers,
		bytes.NewBuffer(marshalledBlock),
		nil,
	)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	proposeResponse, err := validatorClient.proposeBeaconBlock(genericSignedBlock)
	assert.NoError(t, err)
	require.NotNil(t, proposeResponse)

	expectedBlockRoot, err := phase0Block.Phase0.Block.HashTreeRoot()
	require.NoError(t, err)

	// Make sure that the block root is set
	assert.DeepEqual(t, expectedBlockRoot[:], proposeResponse.BlockRoot)
}

func generateSignedPhase0Block() *ethpb.GenericSignedBeaconBlock_Phase0 {
	return &ethpb.GenericSignedBeaconBlock_Phase0{
		Phase0: &ethpb.SignedBeaconBlock{
			Block: &ethpb.BeaconBlock{
				Slot:          1,
				ProposerIndex: 2,
				ParentRoot:    test_helpers.FillByteSlice(32, 3),
				StateRoot:     test_helpers.FillByteSlice(32, 4),
				Body: &ethpb.BeaconBlockBody{
					RandaoReveal: test_helpers.FillByteSlice(96, 5),
					Eth1Data: &ethpb.Eth1Data{
						DepositRoot:  test_helpers.FillByteSlice(32, 6),
						DepositCount: 7,
						BlockHash:    test_helpers.FillByteSlice(32, 8),
					},
					Graffiti: test_helpers.FillByteSlice(32, 9),
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          10,
									ProposerIndex: 11,
									ParentRoot:    test_helpers.FillByteSlice(32, 12),
									StateRoot:     test_helpers.FillByteSlice(32, 13),
									BodyRoot:      test_helpers.FillByteSlice(32, 14),
								},
								Signature: test_helpers.FillByteSlice(96, 15),
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          16,
									ProposerIndex: 17,
									ParentRoot:    test_helpers.FillByteSlice(32, 18),
									StateRoot:     test_helpers.FillByteSlice(32, 19),
									BodyRoot:      test_helpers.FillByteSlice(32, 20),
								},
								Signature: test_helpers.FillByteSlice(96, 21),
							},
						},
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          22,
									ProposerIndex: 23,
									ParentRoot:    test_helpers.FillByteSlice(32, 24),
									StateRoot:     test_helpers.FillByteSlice(32, 25),
									BodyRoot:      test_helpers.FillByteSlice(32, 26),
								},
								Signature: test_helpers.FillByteSlice(96, 27),
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          28,
									ProposerIndex: 29,
									ParentRoot:    test_helpers.FillByteSlice(32, 30),
									StateRoot:     test_helpers.FillByteSlice(32, 31),
									BodyRoot:      test_helpers.FillByteSlice(32, 32),
								},
								Signature: test_helpers.FillByteSlice(96, 33),
							},
						},
					},
					AttesterSlashings: []*ethpb.AttesterSlashing{
						{
							Attestation_1: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{34, 35},
								Data: &ethpb.AttestationData{
									Slot:            36,
									CommitteeIndex:  37,
									BeaconBlockRoot: test_helpers.FillByteSlice(32, 38),
									Source: &ethpb.Checkpoint{
										Epoch: 39,
										Root:  test_helpers.FillByteSlice(32, 40),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 41,
										Root:  test_helpers.FillByteSlice(32, 42),
									},
								},
								Signature: test_helpers.FillByteSlice(96, 43),
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{44, 45},
								Data: &ethpb.AttestationData{
									Slot:            46,
									CommitteeIndex:  47,
									BeaconBlockRoot: test_helpers.FillByteSlice(32, 48),
									Source: &ethpb.Checkpoint{
										Epoch: 49,
										Root:  test_helpers.FillByteSlice(32, 50),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 51,
										Root:  test_helpers.FillByteSlice(32, 52),
									},
								},
								Signature: test_helpers.FillByteSlice(96, 53),
							},
						},
						{
							Attestation_1: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{54, 55},
								Data: &ethpb.AttestationData{
									Slot:            56,
									CommitteeIndex:  57,
									BeaconBlockRoot: test_helpers.FillByteSlice(32, 58),
									Source: &ethpb.Checkpoint{
										Epoch: 59,
										Root:  test_helpers.FillByteSlice(32, 60),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 61,
										Root:  test_helpers.FillByteSlice(32, 62),
									},
								},
								Signature: test_helpers.FillByteSlice(96, 63),
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{64, 65},
								Data: &ethpb.AttestationData{
									Slot:            66,
									CommitteeIndex:  67,
									BeaconBlockRoot: test_helpers.FillByteSlice(32, 68),
									Source: &ethpb.Checkpoint{
										Epoch: 69,
										Root:  test_helpers.FillByteSlice(32, 70),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 71,
										Root:  test_helpers.FillByteSlice(32, 72),
									},
								},
								Signature: test_helpers.FillByteSlice(96, 73),
							},
						},
					},
					Attestations: []*ethpb.Attestation{
						{
							AggregationBits: test_helpers.FillByteSlice(32, 74),
							Data: &ethpb.AttestationData{
								Slot:            75,
								CommitteeIndex:  76,
								BeaconBlockRoot: test_helpers.FillByteSlice(32, 77),
								Source: &ethpb.Checkpoint{
									Epoch: 78,
									Root:  test_helpers.FillByteSlice(32, 79),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 80,
									Root:  test_helpers.FillByteSlice(32, 81),
								},
							},
							Signature: test_helpers.FillByteSlice(96, 82),
						},
						{
							AggregationBits: test_helpers.FillByteSlice(4, 83),
							Data: &ethpb.AttestationData{
								Slot:            84,
								CommitteeIndex:  85,
								BeaconBlockRoot: test_helpers.FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 87,
									Root:  test_helpers.FillByteSlice(32, 88),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 89,
									Root:  test_helpers.FillByteSlice(32, 90),
								},
							},
							Signature: test_helpers.FillByteSlice(96, 91),
						},
					},
					Deposits: []*ethpb.Deposit{
						{
							Proof: test_helpers.FillByteArraySlice(33, test_helpers.FillByteSlice(32, 92)),
							Data: &ethpb.Deposit_Data{
								PublicKey:             test_helpers.FillByteSlice(48, 94),
								WithdrawalCredentials: test_helpers.FillByteSlice(32, 95),
								Amount:                96,
								Signature:             test_helpers.FillByteSlice(96, 97),
							},
						},
						{
							Proof: test_helpers.FillByteArraySlice(33, test_helpers.FillByteSlice(32, 98)),
							Data: &ethpb.Deposit_Data{
								PublicKey:             test_helpers.FillByteSlice(48, 100),
								WithdrawalCredentials: test_helpers.FillByteSlice(32, 101),
								Amount:                102,
								Signature:             test_helpers.FillByteSlice(96, 103),
							},
						},
					},
					VoluntaryExits: []*ethpb.SignedVoluntaryExit{
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          104,
								ValidatorIndex: 105,
							},
							Signature: test_helpers.FillByteSlice(96, 106),
						},
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          107,
								ValidatorIndex: 108,
							},
							Signature: test_helpers.FillByteSlice(96, 109),
						},
					},
				},
			},
			Signature: test_helpers.FillByteSlice(96, 110),
		},
	}
}
