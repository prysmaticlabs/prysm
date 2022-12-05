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
					BaseFeePerGas: littleEndianBytesToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BaseFeePerGas),
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

	// Make sure that the block root is set
	assert.DeepEqual(t, bellatrixBlock.Bellatrix.Block.Body.Attestations[0].Data.BeaconBlockRoot, proposeResponse.BlockRoot)
}

func generateSignedBellatrixBlock() *ethpb.GenericSignedBeaconBlock_Bellatrix {
	return &ethpb.GenericSignedBeaconBlock_Bellatrix{
		Bellatrix: &ethpb.SignedBeaconBlockBellatrix{
			Block: &ethpb.BeaconBlockBellatrix{
				Slot:          1,
				ProposerIndex: 2,
				ParentRoot:    []byte{3},
				StateRoot:     []byte{4},
				Body: &ethpb.BeaconBlockBodyBellatrix{
					RandaoReveal: []byte{5},
					Eth1Data: &ethpb.Eth1Data{
						DepositRoot:  []byte{6},
						DepositCount: 7,
						BlockHash:    []byte{8},
					},
					Graffiti: []byte{9},
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          10,
									ProposerIndex: 11,
									ParentRoot:    []byte{12},
									StateRoot:     []byte{13},
									BodyRoot:      []byte{14},
								},
								Signature: []byte{15},
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          16,
									ProposerIndex: 17,
									ParentRoot:    []byte{18},
									StateRoot:     []byte{19},
									BodyRoot:      []byte{20},
								},
								Signature: []byte{21},
							},
						},
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          22,
									ProposerIndex: 23,
									ParentRoot:    []byte{24},
									StateRoot:     []byte{25},
									BodyRoot:      []byte{26},
								},
								Signature: []byte{27},
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          28,
									ProposerIndex: 29,
									ParentRoot:    []byte{30},
									StateRoot:     []byte{31},
									BodyRoot:      []byte{32},
								},
								Signature: []byte{33},
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
									BeaconBlockRoot: []byte{38},
									Source: &ethpb.Checkpoint{
										Epoch: 39,
										Root:  []byte{40},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 41,
										Root:  []byte{42},
									},
								},
								Signature: []byte{43},
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{44, 45},
								Data: &ethpb.AttestationData{
									Slot:            46,
									CommitteeIndex:  47,
									BeaconBlockRoot: []byte{38},
									Source: &ethpb.Checkpoint{
										Epoch: 49,
										Root:  []byte{50},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 51,
										Root:  []byte{52},
									},
								},
								Signature: []byte{53},
							},
						},
						{
							Attestation_1: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{54, 55},
								Data: &ethpb.AttestationData{
									Slot:            56,
									CommitteeIndex:  57,
									BeaconBlockRoot: []byte{38},
									Source: &ethpb.Checkpoint{
										Epoch: 59,
										Root:  []byte{60},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 61,
										Root:  []byte{62},
									},
								},
								Signature: []byte{63},
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{64, 65},
								Data: &ethpb.AttestationData{
									Slot:            66,
									CommitteeIndex:  67,
									BeaconBlockRoot: []byte{38},
									Source: &ethpb.Checkpoint{
										Epoch: 69,
										Root:  []byte{70},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 71,
										Root:  []byte{72},
									},
								},
								Signature: []byte{73},
							},
						},
					},
					Attestations: []*ethpb.Attestation{
						{
							AggregationBits: []byte{74},
							Data: &ethpb.AttestationData{
								Slot:            75,
								CommitteeIndex:  76,
								BeaconBlockRoot: []byte{38},
								Source: &ethpb.Checkpoint{
									Epoch: 78,
									Root:  []byte{79},
								},
								Target: &ethpb.Checkpoint{
									Epoch: 80,
									Root:  []byte{81},
								},
							},
							Signature: []byte{82},
						},
						{
							AggregationBits: []byte{83},
							Data: &ethpb.AttestationData{
								Slot:            84,
								CommitteeIndex:  85,
								BeaconBlockRoot: []byte{38},
								Source: &ethpb.Checkpoint{
									Epoch: 87,
									Root:  []byte{88},
								},
								Target: &ethpb.Checkpoint{
									Epoch: 89,
									Root:  []byte{90},
								},
							},
							Signature: []byte{91},
						},
					},
					Deposits: []*ethpb.Deposit{
						{
							Proof: [][]byte{
								{92},
								{93},
							},
							Data: &ethpb.Deposit_Data{
								PublicKey:             []byte{94},
								WithdrawalCredentials: []byte{95},
								Amount:                96,
								Signature:             []byte{97},
							},
						},
						{
							Proof: [][]byte{
								{98},
								{99},
							},
							Data: &ethpb.Deposit_Data{
								PublicKey:             []byte{100},
								WithdrawalCredentials: []byte{101},
								Amount:                102,
								Signature:             []byte{103},
							},
						},
					},
					VoluntaryExits: []*ethpb.SignedVoluntaryExit{
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          104,
								ValidatorIndex: 105,
							},
							Signature: []byte{106},
						},
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          107,
								ValidatorIndex: 108,
							},
							Signature: []byte{109},
						},
					},
					SyncAggregate: &ethpb.SyncAggregate{
						SyncCommitteeBits:      []byte{110},
						SyncCommitteeSignature: []byte{111},
					},
					ExecutionPayload: &enginev1.ExecutionPayload{
						ParentHash:    []byte{112},
						FeeRecipient:  []byte{113},
						StateRoot:     []byte{114},
						ReceiptsRoot:  []byte{115},
						LogsBloom:     []byte{116},
						PrevRandao:    []byte{117},
						BlockNumber:   118,
						GasLimit:      119,
						GasUsed:       120,
						Timestamp:     121,
						ExtraData:     []byte{122},
						BaseFeePerGas: []byte{123},
						BlockHash:     []byte{124},
						Transactions: [][]byte{
							{125},
							{126},
						},
					},
				},
			},
			Signature: []byte{127},
		},
	}
}
