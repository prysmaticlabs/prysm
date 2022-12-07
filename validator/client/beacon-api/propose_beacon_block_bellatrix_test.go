//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api/mock"
)

func TestProposeBeaconBlock_Bellatrix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	bellatrixBlock := generateSignedBellatrixBlock()

	genericSignedBlock := &ethpb.GenericSignedBeaconBlock{}
	genericSignedBlock.Block = bellatrixBlock

	jsonBellatrixBlock := &apimiddleware.SignedBeaconBlockBellatrixContainerJson{
		Signature: hexutil.Encode(bellatrixBlock.Bellatrix.Signature),
		Message: &apimiddleware.BeaconBlockBellatrixJson{
			ParentRoot:    hexutil.Encode(bellatrixBlock.Bellatrix.Block.ParentRoot),
			ProposerIndex: uint64ToString(bellatrixBlock.Bellatrix.Block.ProposerIndex),
			Slot:          uint64ToString(bellatrixBlock.Bellatrix.Block.Slot),
			StateRoot:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.StateRoot),
			Body: &apimiddleware.BeaconBlockBodyBellatrixJson{
				Attestations:      jsonifyAttestations(bellatrixBlock.Bellatrix.Block.Body.Attestations),
				AttesterSlashings: jsonifyAttesterSlashings(bellatrixBlock.Bellatrix.Block.Body.AttesterSlashings),
				Deposits:          jsonifyDeposits(bellatrixBlock.Bellatrix.Block.Body.Deposits),
				Eth1Data:          jsonifyEth1Data(bellatrixBlock.Bellatrix.Block.Body.Eth1Data),
				Graffiti:          hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.Graffiti),
				ProposerSlashings: jsonifyProposerSlashings(bellatrixBlock.Bellatrix.Block.Body.ProposerSlashings),
				RandaoReveal:      hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.RandaoReveal),
				VoluntaryExits:    jsonifySignedVoluntaryExits(bellatrixBlock.Bellatrix.Block.Body.VoluntaryExits),
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.SyncAggregate.SyncCommitteeSignature),
				},
				ExecutionPayload: &apimiddleware.ExecutionPayloadJson{
					BaseFeePerGas: bytesutil.LittleEndianBytesToBigInt(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BaseFeePerGas).String(),
					BlockHash:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BlockHash),
					BlockNumber:   uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BlockNumber),
					ExtraData:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.ExtraData),
					FeeRecipient:  hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.FeeRecipient),
					GasLimit:      uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.GasLimit),
					GasUsed:       uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.GasUsed),
					LogsBloom:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.LogsBloom),
					ParentHash:    hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.ParentHash),
					PrevRandao:    hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.PrevRandao),
					ReceiptsRoot:  hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.ReceiptsRoot),
					StateRoot:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.StateRoot),
					TimeStamp:     uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.Timestamp),
					Transactions:  jsonifyTransactions(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.Transactions),
				},
			},
		},
	}

	marshalledBlock, err := json.Marshal(jsonBellatrixBlock)
	require.NoError(t, err)

	// Make sure that what we send in the POST body is the marshalled version of the protobuf block
	headers := map[string]string{"Eth-Consensus-Version": "bellatrix"}
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

	expectedBlockRoot, err := bellatrixBlock.Bellatrix.Block.HashTreeRoot()
	require.NoError(t, err)

	// Make sure that the block root is set
	assert.DeepEqual(t, expectedBlockRoot[:], proposeResponse.BlockRoot)
}

func generateSignedBellatrixBlock() *ethpb.GenericSignedBeaconBlock_Bellatrix {
	return &ethpb.GenericSignedBeaconBlock_Bellatrix{
		Bellatrix: &ethpb.SignedBeaconBlockBellatrix{
			Block: &ethpb.BeaconBlockBellatrix{
				Slot:          1,
				ProposerIndex: 2,
				ParentRoot:    fillByteSlice(32, 3),
				StateRoot:     fillByteSlice(32, 4),
				Body: &ethpb.BeaconBlockBodyBellatrix{
					RandaoReveal: fillByteSlice(96, 5),
					Eth1Data: &ethpb.Eth1Data{
						DepositRoot:  fillByteSlice(32, 6),
						DepositCount: 7,
						BlockHash:    fillByteSlice(32, 8),
					},
					Graffiti: fillByteSlice(32, 9),
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          10,
									ProposerIndex: 11,
									ParentRoot:    fillByteSlice(32, 12),
									StateRoot:     fillByteSlice(32, 13),
									BodyRoot:      fillByteSlice(32, 14),
								},
								Signature: fillByteSlice(96, 15),
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          16,
									ProposerIndex: 17,
									ParentRoot:    fillByteSlice(32, 18),
									StateRoot:     fillByteSlice(32, 19),
									BodyRoot:      fillByteSlice(32, 20),
								},
								Signature: fillByteSlice(96, 21),
							},
						},
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          22,
									ProposerIndex: 23,
									ParentRoot:    fillByteSlice(32, 24),
									StateRoot:     fillByteSlice(32, 25),
									BodyRoot:      fillByteSlice(32, 26),
								},
								Signature: fillByteSlice(96, 27),
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          28,
									ProposerIndex: 29,
									ParentRoot:    fillByteSlice(32, 30),
									StateRoot:     fillByteSlice(32, 31),
									BodyRoot:      fillByteSlice(32, 32),
								},
								Signature: fillByteSlice(96, 33),
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
									BeaconBlockRoot: fillByteSlice(32, 38),
									Source: &ethpb.Checkpoint{
										Epoch: 39,
										Root:  fillByteSlice(32, 40),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 41,
										Root:  fillByteSlice(32, 42),
									},
								},
								Signature: fillByteSlice(96, 43),
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{44, 45},
								Data: &ethpb.AttestationData{
									Slot:            46,
									CommitteeIndex:  47,
									BeaconBlockRoot: fillByteSlice(32, 38),
									Source: &ethpb.Checkpoint{
										Epoch: 49,
										Root:  fillByteSlice(32, 50),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 51,
										Root:  fillByteSlice(32, 52),
									},
								},
								Signature: fillByteSlice(96, 53),
							},
						},
						{
							Attestation_1: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{54, 55},
								Data: &ethpb.AttestationData{
									Slot:            56,
									CommitteeIndex:  57,
									BeaconBlockRoot: fillByteSlice(32, 38),
									Source: &ethpb.Checkpoint{
										Epoch: 59,
										Root:  fillByteSlice(32, 60),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 61,
										Root:  fillByteSlice(32, 62),
									},
								},
								Signature: fillByteSlice(96, 63),
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{64, 65},
								Data: &ethpb.AttestationData{
									Slot:            66,
									CommitteeIndex:  67,
									BeaconBlockRoot: fillByteSlice(32, 38),
									Source: &ethpb.Checkpoint{
										Epoch: 69,
										Root:  fillByteSlice(32, 70),
									},
									Target: &ethpb.Checkpoint{
										Epoch: 71,
										Root:  fillByteSlice(32, 72),
									},
								},
								Signature: fillByteSlice(96, 73),
							},
						},
					},
					Attestations: []*ethpb.Attestation{
						{
							AggregationBits: fillByteSlice(4, 74),
							Data: &ethpb.AttestationData{
								Slot:            75,
								CommitteeIndex:  76,
								BeaconBlockRoot: fillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 78,
									Root:  fillByteSlice(32, 79),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 80,
									Root:  fillByteSlice(32, 81),
								},
							},
							Signature: fillByteSlice(96, 82),
						},
						{
							AggregationBits: fillByteSlice(4, 83),
							Data: &ethpb.AttestationData{
								Slot:            84,
								CommitteeIndex:  85,
								BeaconBlockRoot: fillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 87,
									Root:  fillByteSlice(32, 88),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 89,
									Root:  fillByteSlice(32, 90),
								},
							},
							Signature: fillByteSlice(96, 91),
						},
					},
					Deposits: []*ethpb.Deposit{
						{
							Proof: fillByteArraySlice(33, fillByteSlice(32, 92)),
							Data: &ethpb.Deposit_Data{
								PublicKey:             fillByteSlice(48, 94),
								WithdrawalCredentials: fillByteSlice(32, 95),
								Amount:                96,
								Signature:             fillByteSlice(96, 97),
							},
						},
						{
							Proof: fillByteArraySlice(33, fillByteSlice(32, 98)),
							Data: &ethpb.Deposit_Data{
								PublicKey:             fillByteSlice(48, 100),
								WithdrawalCredentials: fillByteSlice(32, 101),
								Amount:                102,
								Signature:             fillByteSlice(96, 103),
							},
						},
					},
					VoluntaryExits: []*ethpb.SignedVoluntaryExit{
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          104,
								ValidatorIndex: 105,
							},
							Signature: fillByteSlice(96, 106),
						},
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          107,
								ValidatorIndex: 108,
							},
							Signature: fillByteSlice(96, 109),
						},
					},
					SyncAggregate: &ethpb.SyncAggregate{
						SyncCommitteeBits:      fillByteSlice(64, 110),
						SyncCommitteeSignature: fillByteSlice(96, 111),
					},
					ExecutionPayload: &enginev1.ExecutionPayload{
						ParentHash:    fillByteSlice(32, 112),
						FeeRecipient:  fillByteSlice(20, 113),
						StateRoot:     fillByteSlice(32, 114),
						ReceiptsRoot:  fillByteSlice(32, 115),
						LogsBloom:     fillByteSlice(256, 116),
						PrevRandao:    fillByteSlice(32, 117),
						BlockNumber:   118,
						GasLimit:      119,
						GasUsed:       120,
						Timestamp:     121,
						ExtraData:     fillByteSlice(32, 122),
						BaseFeePerGas: fillByteSlice(32, 123),
						BlockHash:     fillByteSlice(32, 124),
						Transactions: [][]byte{
							{125},
							{126},
						},
					},
				},
			},
			Signature: fillByteSlice(96, 127),
		},
	}
}
