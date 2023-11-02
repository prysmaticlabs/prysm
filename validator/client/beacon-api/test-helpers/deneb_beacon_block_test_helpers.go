package test_helpers

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

func GenerateProtoDenebBeaconBlock() *ethpb.BeaconBlockAndBlobsDeneb {
	return &ethpb.BeaconBlockAndBlobsDeneb{
		Block: &ethpb.BeaconBlockDeneb{
			Slot:          1,
			ProposerIndex: 2,
			ParentRoot:    FillByteSlice(32, 3),
			StateRoot:     FillByteSlice(32, 4),
			Body: &ethpb.BeaconBlockBodyDeneb{
				RandaoReveal: FillByteSlice(96, 5),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  FillByteSlice(32, 6),
					DepositCount: 7,
					BlockHash:    FillByteSlice(32, 8),
				},
				Graffiti: FillByteSlice(32, 9),
				ProposerSlashings: []*ethpb.ProposerSlashing{
					{
						Header_1: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          10,
								ProposerIndex: 11,
								ParentRoot:    FillByteSlice(32, 12),
								StateRoot:     FillByteSlice(32, 13),
								BodyRoot:      FillByteSlice(32, 14),
							},
							Signature: FillByteSlice(96, 15),
						},
						Header_2: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          16,
								ProposerIndex: 17,
								ParentRoot:    FillByteSlice(32, 18),
								StateRoot:     FillByteSlice(32, 19),
								BodyRoot:      FillByteSlice(32, 20),
							},
							Signature: FillByteSlice(96, 21),
						},
					},
					{
						Header_1: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          22,
								ProposerIndex: 23,
								ParentRoot:    FillByteSlice(32, 24),
								StateRoot:     FillByteSlice(32, 25),
								BodyRoot:      FillByteSlice(32, 26),
							},
							Signature: FillByteSlice(96, 27),
						},
						Header_2: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          28,
								ProposerIndex: 29,
								ParentRoot:    FillByteSlice(32, 30),
								StateRoot:     FillByteSlice(32, 31),
								BodyRoot:      FillByteSlice(32, 32),
							},
							Signature: FillByteSlice(96, 33),
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
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 39,
									Root:  FillByteSlice(32, 40),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 41,
									Root:  FillByteSlice(32, 42),
								},
							},
							Signature: FillByteSlice(96, 43),
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{44, 45},
							Data: &ethpb.AttestationData{
								Slot:            46,
								CommitteeIndex:  47,
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 49,
									Root:  FillByteSlice(32, 50),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 51,
									Root:  FillByteSlice(32, 52),
								},
							},
							Signature: FillByteSlice(96, 53),
						},
					},
					{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{54, 55},
							Data: &ethpb.AttestationData{
								Slot:            56,
								CommitteeIndex:  57,
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 59,
									Root:  FillByteSlice(32, 60),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 61,
									Root:  FillByteSlice(32, 62),
								},
							},
							Signature: FillByteSlice(96, 63),
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{64, 65},
							Data: &ethpb.AttestationData{
								Slot:            66,
								CommitteeIndex:  67,
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 69,
									Root:  FillByteSlice(32, 70),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 71,
									Root:  FillByteSlice(32, 72),
								},
							},
							Signature: FillByteSlice(96, 73),
						},
					},
				},
				Attestations: []*ethpb.Attestation{
					{
						AggregationBits: FillByteSlice(4, 74),
						Data: &ethpb.AttestationData{
							Slot:            75,
							CommitteeIndex:  76,
							BeaconBlockRoot: FillByteSlice(32, 38),
							Source: &ethpb.Checkpoint{
								Epoch: 78,
								Root:  FillByteSlice(32, 79),
							},
							Target: &ethpb.Checkpoint{
								Epoch: 80,
								Root:  FillByteSlice(32, 81),
							},
						},
						Signature: FillByteSlice(96, 82),
					},
					{
						AggregationBits: FillByteSlice(4, 83),
						Data: &ethpb.AttestationData{
							Slot:            84,
							CommitteeIndex:  85,
							BeaconBlockRoot: FillByteSlice(32, 38),
							Source: &ethpb.Checkpoint{
								Epoch: 87,
								Root:  FillByteSlice(32, 88),
							},
							Target: &ethpb.Checkpoint{
								Epoch: 89,
								Root:  FillByteSlice(32, 90),
							},
						},
						Signature: FillByteSlice(96, 91),
					},
				},
				Deposits: []*ethpb.Deposit{
					{
						Proof: FillByteArraySlice(33, FillByteSlice(32, 92)),
						Data: &ethpb.Deposit_Data{
							PublicKey:             FillByteSlice(48, 94),
							WithdrawalCredentials: FillByteSlice(32, 95),
							Amount:                96,
							Signature:             FillByteSlice(96, 97),
						},
					},
					{
						Proof: FillByteArraySlice(33, FillByteSlice(32, 98)),
						Data: &ethpb.Deposit_Data{
							PublicKey:             FillByteSlice(48, 100),
							WithdrawalCredentials: FillByteSlice(32, 101),
							Amount:                102,
							Signature:             FillByteSlice(96, 103),
						},
					},
				},
				VoluntaryExits: []*ethpb.SignedVoluntaryExit{
					{
						Exit: &ethpb.VoluntaryExit{
							Epoch:          104,
							ValidatorIndex: 105,
						},
						Signature: FillByteSlice(96, 106),
					},
					{
						Exit: &ethpb.VoluntaryExit{
							Epoch:          107,
							ValidatorIndex: 108,
						},
						Signature: FillByteSlice(96, 109),
					},
				},
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      FillByteSlice(64, 110),
					SyncCommitteeSignature: FillByteSlice(96, 111),
				},
				ExecutionPayload: &enginev1.ExecutionPayloadDeneb{
					ParentHash:    FillByteSlice(32, 112),
					FeeRecipient:  FillByteSlice(20, 113),
					StateRoot:     FillByteSlice(32, 114),
					ReceiptsRoot:  FillByteSlice(32, 115),
					LogsBloom:     FillByteSlice(256, 116),
					PrevRandao:    FillByteSlice(32, 117),
					BlockNumber:   118,
					GasLimit:      119,
					GasUsed:       120,
					Timestamp:     121,
					ExtraData:     FillByteSlice(32, 122),
					BaseFeePerGas: FillByteSlice(32, 123),
					BlockHash:     FillByteSlice(32, 124),
					Transactions: [][]byte{
						FillByteSlice(32, 125),
						FillByteSlice(32, 126),
					},
					Withdrawals: []*enginev1.Withdrawal{
						{
							Index:          127,
							ValidatorIndex: 128,
							Address:        FillByteSlice(20, 129),
							Amount:         130,
						},
						{
							Index:          131,
							ValidatorIndex: 132,
							Address:        FillByteSlice(20, 133),
							Amount:         134,
						},
					},
					BlobGasUsed:   135,
					ExcessBlobGas: 136,
				},
				BlsToExecutionChanges: []*ethpb.SignedBLSToExecutionChange{
					{
						Message: &ethpb.BLSToExecutionChange{
							ValidatorIndex:     137,
							FromBlsPubkey:      FillByteSlice(48, 138),
							ToExecutionAddress: FillByteSlice(20, 139),
						},
						Signature: FillByteSlice(96, 140),
					},
					{
						Message: &ethpb.BLSToExecutionChange{
							ValidatorIndex:     141,
							FromBlsPubkey:      FillByteSlice(48, 142),
							ToExecutionAddress: FillByteSlice(20, 143),
						},
						Signature: FillByteSlice(96, 144),
					},
				},
				BlobKzgCommitments: [][]byte{FillByteSlice(48, 145), FillByteSlice(48, 146)},
			},
		},
		Blobs: []*ethpb.DeprecatedBlobSidecar{
			{
				BlockRoot:       FillByteSlice(32, 147),
				Index:           148,
				Slot:            149,
				BlockParentRoot: FillByteSlice(32, 150),
				ProposerIndex:   151,
				Blob:            FillByteSlice(131072, 152),
				KzgCommitment:   FillByteSlice(48, 153),
				KzgProof:        FillByteSlice(48, 154),
			},
			{
				BlockRoot:       FillByteSlice(32, 155),
				Index:           156,
				Slot:            157,
				BlockParentRoot: FillByteSlice(32, 158),
				ProposerIndex:   159,
				Blob:            FillByteSlice(131072, 160),
				KzgCommitment:   FillByteSlice(48, 161),
				KzgProof:        FillByteSlice(48, 162),
			},
		},
	}
}

func GenerateProtoBlindedDenebBeaconBlock() *ethpb.BlindedBeaconBlockAndBlobsDeneb {
	return &ethpb.BlindedBeaconBlockAndBlobsDeneb{
		Block: &ethpb.BlindedBeaconBlockDeneb{
			Slot:          1,
			ProposerIndex: 2,
			ParentRoot:    FillByteSlice(32, 3),
			StateRoot:     FillByteSlice(32, 4),
			Body: &ethpb.BlindedBeaconBlockBodyDeneb{
				RandaoReveal: FillByteSlice(96, 5),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  FillByteSlice(32, 6),
					DepositCount: 7,
					BlockHash:    FillByteSlice(32, 8),
				},
				Graffiti: FillByteSlice(32, 9),
				ProposerSlashings: []*ethpb.ProposerSlashing{
					{
						Header_1: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          10,
								ProposerIndex: 11,
								ParentRoot:    FillByteSlice(32, 12),
								StateRoot:     FillByteSlice(32, 13),
								BodyRoot:      FillByteSlice(32, 14),
							},
							Signature: FillByteSlice(96, 15),
						},
						Header_2: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          16,
								ProposerIndex: 17,
								ParentRoot:    FillByteSlice(32, 18),
								StateRoot:     FillByteSlice(32, 19),
								BodyRoot:      FillByteSlice(32, 20),
							},
							Signature: FillByteSlice(96, 21),
						},
					},
					{
						Header_1: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          22,
								ProposerIndex: 23,
								ParentRoot:    FillByteSlice(32, 24),
								StateRoot:     FillByteSlice(32, 25),
								BodyRoot:      FillByteSlice(32, 26),
							},
							Signature: FillByteSlice(96, 27),
						},
						Header_2: &ethpb.SignedBeaconBlockHeader{
							Header: &ethpb.BeaconBlockHeader{
								Slot:          28,
								ProposerIndex: 29,
								ParentRoot:    FillByteSlice(32, 30),
								StateRoot:     FillByteSlice(32, 31),
								BodyRoot:      FillByteSlice(32, 32),
							},
							Signature: FillByteSlice(96, 33),
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
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 39,
									Root:  FillByteSlice(32, 40),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 41,
									Root:  FillByteSlice(32, 42),
								},
							},
							Signature: FillByteSlice(96, 43),
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{44, 45},
							Data: &ethpb.AttestationData{
								Slot:            46,
								CommitteeIndex:  47,
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 49,
									Root:  FillByteSlice(32, 50),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 51,
									Root:  FillByteSlice(32, 52),
								},
							},
							Signature: FillByteSlice(96, 53),
						},
					},
					{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{54, 55},
							Data: &ethpb.AttestationData{
								Slot:            56,
								CommitteeIndex:  57,
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 59,
									Root:  FillByteSlice(32, 60),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 61,
									Root:  FillByteSlice(32, 62),
								},
							},
							Signature: FillByteSlice(96, 63),
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{64, 65},
							Data: &ethpb.AttestationData{
								Slot:            66,
								CommitteeIndex:  67,
								BeaconBlockRoot: FillByteSlice(32, 38),
								Source: &ethpb.Checkpoint{
									Epoch: 69,
									Root:  FillByteSlice(32, 70),
								},
								Target: &ethpb.Checkpoint{
									Epoch: 71,
									Root:  FillByteSlice(32, 72),
								},
							},
							Signature: FillByteSlice(96, 73),
						},
					},
				},
				Attestations: []*ethpb.Attestation{
					{
						AggregationBits: FillByteSlice(4, 74),
						Data: &ethpb.AttestationData{
							Slot:            75,
							CommitteeIndex:  76,
							BeaconBlockRoot: FillByteSlice(32, 38),
							Source: &ethpb.Checkpoint{
								Epoch: 78,
								Root:  FillByteSlice(32, 79),
							},
							Target: &ethpb.Checkpoint{
								Epoch: 80,
								Root:  FillByteSlice(32, 81),
							},
						},
						Signature: FillByteSlice(96, 82),
					},
					{
						AggregationBits: FillByteSlice(4, 83),
						Data: &ethpb.AttestationData{
							Slot:            84,
							CommitteeIndex:  85,
							BeaconBlockRoot: FillByteSlice(32, 38),
							Source: &ethpb.Checkpoint{
								Epoch: 87,
								Root:  FillByteSlice(32, 88),
							},
							Target: &ethpb.Checkpoint{
								Epoch: 89,
								Root:  FillByteSlice(32, 90),
							},
						},
						Signature: FillByteSlice(96, 91),
					},
				},
				Deposits: []*ethpb.Deposit{
					{
						Proof: FillByteArraySlice(33, FillByteSlice(32, 92)),
						Data: &ethpb.Deposit_Data{
							PublicKey:             FillByteSlice(48, 94),
							WithdrawalCredentials: FillByteSlice(32, 95),
							Amount:                96,
							Signature:             FillByteSlice(96, 97),
						},
					},
					{
						Proof: FillByteArraySlice(33, FillByteSlice(32, 98)),
						Data: &ethpb.Deposit_Data{
							PublicKey:             FillByteSlice(48, 100),
							WithdrawalCredentials: FillByteSlice(32, 101),
							Amount:                102,
							Signature:             FillByteSlice(96, 103),
						},
					},
				},
				VoluntaryExits: []*ethpb.SignedVoluntaryExit{
					{
						Exit: &ethpb.VoluntaryExit{
							Epoch:          104,
							ValidatorIndex: 105,
						},
						Signature: FillByteSlice(96, 106),
					},
					{
						Exit: &ethpb.VoluntaryExit{
							Epoch:          107,
							ValidatorIndex: 108,
						},
						Signature: FillByteSlice(96, 109),
					},
				},
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      FillByteSlice(64, 110),
					SyncCommitteeSignature: FillByteSlice(96, 111),
				},
				ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderDeneb{
					ParentHash:       FillByteSlice(32, 112),
					FeeRecipient:     FillByteSlice(20, 113),
					StateRoot:        FillByteSlice(32, 114),
					ReceiptsRoot:     FillByteSlice(32, 115),
					LogsBloom:        FillByteSlice(256, 116),
					PrevRandao:       FillByteSlice(32, 117),
					BlockNumber:      118,
					GasLimit:         119,
					GasUsed:          120,
					Timestamp:        121,
					ExtraData:        FillByteSlice(32, 122),
					BaseFeePerGas:    FillByteSlice(32, 123),
					BlockHash:        FillByteSlice(32, 124),
					TransactionsRoot: FillByteSlice(32, 125),
					WithdrawalsRoot:  FillByteSlice(32, 126),
					BlobGasUsed:      127,
					ExcessBlobGas:    128,
				},
				BlsToExecutionChanges: []*ethpb.SignedBLSToExecutionChange{
					{
						Message: &ethpb.BLSToExecutionChange{
							ValidatorIndex:     129,
							FromBlsPubkey:      FillByteSlice(48, 130),
							ToExecutionAddress: FillByteSlice(20, 131),
						},
						Signature: FillByteSlice(96, 132),
					},
					{
						Message: &ethpb.BLSToExecutionChange{
							ValidatorIndex:     133,
							FromBlsPubkey:      FillByteSlice(48, 134),
							ToExecutionAddress: FillByteSlice(20, 135),
						},
						Signature: FillByteSlice(96, 136),
					},
				},
				BlobKzgCommitments: [][]byte{FillByteSlice(48, 137), FillByteSlice(48, 138)},
			},
		},
		Blobs: []*ethpb.BlindedBlobSidecar{
			{
				BlockRoot:       FillByteSlice(32, 139),
				Index:           140,
				Slot:            141,
				BlockParentRoot: FillByteSlice(32, 142),
				ProposerIndex:   143,
				BlobRoot:        FillByteSlice(32, 144),
				KzgCommitment:   FillByteSlice(48, 145),
				KzgProof:        FillByteSlice(48, 146),
			},
			{
				BlockRoot:       FillByteSlice(32, 147),
				Index:           148,
				Slot:            149,
				BlockParentRoot: FillByteSlice(32, 150),
				ProposerIndex:   151,
				BlobRoot:        FillByteSlice(32, 152),
				KzgCommitment:   FillByteSlice(48, 153),
				KzgProof:        FillByteSlice(48, 154),
			},
		},
	}
}

func GenerateJsonDenebBeaconBlock() *apimiddleware.BeaconBlockContentsDenebJson {
	return &apimiddleware.BeaconBlockContentsDenebJson{
		Block: &apimiddleware.BeaconBlockDenebJson{
			Slot:          "1",
			ProposerIndex: "2",
			ParentRoot:    FillEncodedByteSlice(32, 3),
			StateRoot:     FillEncodedByteSlice(32, 4),
			Body: &apimiddleware.BeaconBlockBodyDenebJson{
				RandaoReveal: FillEncodedByteSlice(96, 5),
				Eth1Data: &apimiddleware.Eth1DataJson{
					DepositRoot:  FillEncodedByteSlice(32, 6),
					DepositCount: "7",
					BlockHash:    FillEncodedByteSlice(32, 8),
				},
				Graffiti: FillEncodedByteSlice(32, 9),
				ProposerSlashings: []*apimiddleware.ProposerSlashingJson{
					{
						Header_1: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "10",
								ProposerIndex: "11",
								ParentRoot:    FillEncodedByteSlice(32, 12),
								StateRoot:     FillEncodedByteSlice(32, 13),
								BodyRoot:      FillEncodedByteSlice(32, 14),
							},
							Signature: FillEncodedByteSlice(96, 15),
						},
						Header_2: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "16",
								ProposerIndex: "17",
								ParentRoot:    FillEncodedByteSlice(32, 18),
								StateRoot:     FillEncodedByteSlice(32, 19),
								BodyRoot:      FillEncodedByteSlice(32, 20),
							},
							Signature: FillEncodedByteSlice(96, 21),
						},
					},
					{
						Header_1: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "22",
								ProposerIndex: "23",
								ParentRoot:    FillEncodedByteSlice(32, 24),
								StateRoot:     FillEncodedByteSlice(32, 25),
								BodyRoot:      FillEncodedByteSlice(32, 26),
							},
							Signature: FillEncodedByteSlice(96, 27),
						},
						Header_2: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "28",
								ProposerIndex: "29",
								ParentRoot:    FillEncodedByteSlice(32, 30),
								StateRoot:     FillEncodedByteSlice(32, 31),
								BodyRoot:      FillEncodedByteSlice(32, 32),
							},
							Signature: FillEncodedByteSlice(96, 33),
						},
					},
				},
				AttesterSlashings: []*apimiddleware.AttesterSlashingJson{
					{
						Attestation_1: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"34", "35"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "36",
								CommitteeIndex:  "37",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "39",
									Root:  FillEncodedByteSlice(32, 40),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "41",
									Root:  FillEncodedByteSlice(32, 42),
								},
							},
							Signature: FillEncodedByteSlice(96, 43),
						},
						Attestation_2: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"44", "45"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "46",
								CommitteeIndex:  "47",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "49",
									Root:  FillEncodedByteSlice(32, 50),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "51",
									Root:  FillEncodedByteSlice(32, 52),
								},
							},
							Signature: FillEncodedByteSlice(96, 53),
						},
					},
					{
						Attestation_1: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"54", "55"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "56",
								CommitteeIndex:  "57",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "59",
									Root:  FillEncodedByteSlice(32, 60),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "61",
									Root:  FillEncodedByteSlice(32, 62),
								},
							},
							Signature: FillEncodedByteSlice(96, 63),
						},
						Attestation_2: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"64", "65"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "66",
								CommitteeIndex:  "67",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "69",
									Root:  FillEncodedByteSlice(32, 70),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "71",
									Root:  FillEncodedByteSlice(32, 72),
								},
							},
							Signature: FillEncodedByteSlice(96, 73),
						},
					},
				},
				Attestations: []*apimiddleware.AttestationJson{
					{
						AggregationBits: FillEncodedByteSlice(4, 74),
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "75",
							CommitteeIndex:  "76",
							BeaconBlockRoot: FillEncodedByteSlice(32, 38),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "78",
								Root:  FillEncodedByteSlice(32, 79),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "80",
								Root:  FillEncodedByteSlice(32, 81),
							},
						},
						Signature: FillEncodedByteSlice(96, 82),
					},
					{
						AggregationBits: FillEncodedByteSlice(4, 83),
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "84",
							CommitteeIndex:  "85",
							BeaconBlockRoot: FillEncodedByteSlice(32, 38),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "87",
								Root:  FillEncodedByteSlice(32, 88),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "89",
								Root:  FillEncodedByteSlice(32, 90),
							},
						},
						Signature: FillEncodedByteSlice(96, 91),
					},
				},
				Deposits: []*apimiddleware.DepositJson{
					{
						Proof: FillEncodedByteArraySlice(33, FillEncodedByteSlice(32, 92)),
						Data: &apimiddleware.Deposit_DataJson{
							PublicKey:             FillEncodedByteSlice(48, 94),
							WithdrawalCredentials: FillEncodedByteSlice(32, 95),
							Amount:                "96",
							Signature:             FillEncodedByteSlice(96, 97),
						},
					},
					{
						Proof: FillEncodedByteArraySlice(33, FillEncodedByteSlice(32, 98)),
						Data: &apimiddleware.Deposit_DataJson{
							PublicKey:             FillEncodedByteSlice(48, 100),
							WithdrawalCredentials: FillEncodedByteSlice(32, 101),
							Amount:                "102",
							Signature:             FillEncodedByteSlice(96, 103),
						},
					},
				},
				VoluntaryExits: []*apimiddleware.SignedVoluntaryExitJson{
					{
						Exit: &apimiddleware.VoluntaryExitJson{
							Epoch:          "104",
							ValidatorIndex: "105",
						},
						Signature: FillEncodedByteSlice(96, 106),
					},
					{
						Exit: &apimiddleware.VoluntaryExitJson{
							Epoch:          "107",
							ValidatorIndex: "108",
						},
						Signature: FillEncodedByteSlice(96, 109),
					},
				},
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      FillEncodedByteSlice(64, 110),
					SyncCommitteeSignature: FillEncodedByteSlice(96, 111),
				},
				ExecutionPayload: &apimiddleware.ExecutionPayloadDenebJson{
					ParentHash:    FillEncodedByteSlice(32, 112),
					FeeRecipient:  FillEncodedByteSlice(20, 113),
					StateRoot:     FillEncodedByteSlice(32, 114),
					ReceiptsRoot:  FillEncodedByteSlice(32, 115),
					LogsBloom:     FillEncodedByteSlice(256, 116),
					PrevRandao:    FillEncodedByteSlice(32, 117),
					BlockNumber:   "118",
					GasLimit:      "119",
					GasUsed:       "120",
					TimeStamp:     "121",
					ExtraData:     FillEncodedByteSlice(32, 122),
					BaseFeePerGas: bytesutil.LittleEndianBytesToBigInt(FillByteSlice(32, 123)).String(),
					BlockHash:     FillEncodedByteSlice(32, 124),
					Transactions: []string{
						FillEncodedByteSlice(32, 125),
						FillEncodedByteSlice(32, 126),
					},
					Withdrawals: []*apimiddleware.WithdrawalJson{
						{
							WithdrawalIndex:  "127",
							ValidatorIndex:   "128",
							ExecutionAddress: FillEncodedByteSlice(20, 129),
							Amount:           "130",
						},
						{
							WithdrawalIndex:  "131",
							ValidatorIndex:   "132",
							ExecutionAddress: FillEncodedByteSlice(20, 133),
							Amount:           "134",
						},
					},
					BlobGasUsed:   "135",
					ExcessBlobGas: "136",
				},
				BLSToExecutionChanges: []*apimiddleware.SignedBLSToExecutionChangeJson{
					{
						Message: &apimiddleware.BLSToExecutionChangeJson{
							ValidatorIndex:     "137",
							FromBLSPubkey:      FillEncodedByteSlice(48, 138),
							ToExecutionAddress: FillEncodedByteSlice(20, 139),
						},
						Signature: FillEncodedByteSlice(96, 140),
					},
					{
						Message: &apimiddleware.BLSToExecutionChangeJson{
							ValidatorIndex:     "141",
							FromBLSPubkey:      FillEncodedByteSlice(48, 142),
							ToExecutionAddress: FillEncodedByteSlice(20, 143),
						},
						Signature: FillEncodedByteSlice(96, 144),
					},
				},
				BlobKzgCommitments: []string{FillEncodedByteSlice(48, 145), FillEncodedByteSlice(48, 146)},
			},
		},
		BlobSidecars: []*apimiddleware.BlobSidecarJson{
			{
				BlockRoot:       FillEncodedByteSlice(32, 147),
				Index:           "148",
				Slot:            "149",
				BlockParentRoot: FillEncodedByteSlice(32, 150),
				ProposerIndex:   "151",
				Blob:            FillEncodedByteSlice(131072, 152),
				KzgCommitment:   FillEncodedByteSlice(48, 153),
				KzgProof:        FillEncodedByteSlice(48, 154),
			},
			{
				BlockRoot:       FillEncodedByteSlice(32, 155),
				Index:           "156",
				Slot:            "157",
				BlockParentRoot: FillEncodedByteSlice(32, 158),
				ProposerIndex:   "159",
				Blob:            FillEncodedByteSlice(131072, 160),
				KzgCommitment:   FillEncodedByteSlice(48, 161),
				KzgProof:        FillEncodedByteSlice(48, 162),
			},
		},
	}
}

func GenerateJsonBlindedDenebBeaconBlock() *apimiddleware.BlindedBeaconBlockContentsDenebJson {
	return &apimiddleware.BlindedBeaconBlockContentsDenebJson{
		BlindedBlock: &apimiddleware.BlindedBeaconBlockDenebJson{
			Slot:          "1",
			ProposerIndex: "2",
			ParentRoot:    FillEncodedByteSlice(32, 3),
			StateRoot:     FillEncodedByteSlice(32, 4),
			Body: &apimiddleware.BlindedBeaconBlockBodyDenebJson{
				RandaoReveal: FillEncodedByteSlice(96, 5),
				Eth1Data: &apimiddleware.Eth1DataJson{
					DepositRoot:  FillEncodedByteSlice(32, 6),
					DepositCount: "7",
					BlockHash:    FillEncodedByteSlice(32, 8),
				},
				Graffiti: FillEncodedByteSlice(32, 9),
				ProposerSlashings: []*apimiddleware.ProposerSlashingJson{
					{
						Header_1: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "10",
								ProposerIndex: "11",
								ParentRoot:    FillEncodedByteSlice(32, 12),
								StateRoot:     FillEncodedByteSlice(32, 13),
								BodyRoot:      FillEncodedByteSlice(32, 14),
							},
							Signature: FillEncodedByteSlice(96, 15),
						},
						Header_2: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "16",
								ProposerIndex: "17",
								ParentRoot:    FillEncodedByteSlice(32, 18),
								StateRoot:     FillEncodedByteSlice(32, 19),
								BodyRoot:      FillEncodedByteSlice(32, 20),
							},
							Signature: FillEncodedByteSlice(96, 21),
						},
					},
					{
						Header_1: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "22",
								ProposerIndex: "23",
								ParentRoot:    FillEncodedByteSlice(32, 24),
								StateRoot:     FillEncodedByteSlice(32, 25),
								BodyRoot:      FillEncodedByteSlice(32, 26),
							},
							Signature: FillEncodedByteSlice(96, 27),
						},
						Header_2: &apimiddleware.SignedBeaconBlockHeaderJson{
							Header: &apimiddleware.BeaconBlockHeaderJson{
								Slot:          "28",
								ProposerIndex: "29",
								ParentRoot:    FillEncodedByteSlice(32, 30),
								StateRoot:     FillEncodedByteSlice(32, 31),
								BodyRoot:      FillEncodedByteSlice(32, 32),
							},
							Signature: FillEncodedByteSlice(96, 33),
						},
					},
				},
				AttesterSlashings: []*apimiddleware.AttesterSlashingJson{
					{
						Attestation_1: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"34", "35"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "36",
								CommitteeIndex:  "37",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "39",
									Root:  FillEncodedByteSlice(32, 40),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "41",
									Root:  FillEncodedByteSlice(32, 42),
								},
							},
							Signature: FillEncodedByteSlice(96, 43),
						},
						Attestation_2: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"44", "45"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "46",
								CommitteeIndex:  "47",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "49",
									Root:  FillEncodedByteSlice(32, 50),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "51",
									Root:  FillEncodedByteSlice(32, 52),
								},
							},
							Signature: FillEncodedByteSlice(96, 53),
						},
					},
					{
						Attestation_1: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"54", "55"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "56",
								CommitteeIndex:  "57",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "59",
									Root:  FillEncodedByteSlice(32, 60),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "61",
									Root:  FillEncodedByteSlice(32, 62),
								},
							},
							Signature: FillEncodedByteSlice(96, 63),
						},
						Attestation_2: &apimiddleware.IndexedAttestationJson{
							AttestingIndices: []string{"64", "65"},
							Data: &apimiddleware.AttestationDataJson{
								Slot:            "66",
								CommitteeIndex:  "67",
								BeaconBlockRoot: FillEncodedByteSlice(32, 38),
								Source: &apimiddleware.CheckpointJson{
									Epoch: "69",
									Root:  FillEncodedByteSlice(32, 70),
								},
								Target: &apimiddleware.CheckpointJson{
									Epoch: "71",
									Root:  FillEncodedByteSlice(32, 72),
								},
							},
							Signature: FillEncodedByteSlice(96, 73),
						},
					},
				},
				Attestations: []*apimiddleware.AttestationJson{
					{
						AggregationBits: FillEncodedByteSlice(4, 74),
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "75",
							CommitteeIndex:  "76",
							BeaconBlockRoot: FillEncodedByteSlice(32, 38),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "78",
								Root:  FillEncodedByteSlice(32, 79),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "80",
								Root:  FillEncodedByteSlice(32, 81),
							},
						},
						Signature: FillEncodedByteSlice(96, 82),
					},
					{
						AggregationBits: FillEncodedByteSlice(4, 83),
						Data: &apimiddleware.AttestationDataJson{
							Slot:            "84",
							CommitteeIndex:  "85",
							BeaconBlockRoot: FillEncodedByteSlice(32, 38),
							Source: &apimiddleware.CheckpointJson{
								Epoch: "87",
								Root:  FillEncodedByteSlice(32, 88),
							},
							Target: &apimiddleware.CheckpointJson{
								Epoch: "89",
								Root:  FillEncodedByteSlice(32, 90),
							},
						},
						Signature: FillEncodedByteSlice(96, 91),
					},
				},
				Deposits: []*apimiddleware.DepositJson{
					{
						Proof: FillEncodedByteArraySlice(33, FillEncodedByteSlice(32, 92)),
						Data: &apimiddleware.Deposit_DataJson{
							PublicKey:             FillEncodedByteSlice(48, 94),
							WithdrawalCredentials: FillEncodedByteSlice(32, 95),
							Amount:                "96",
							Signature:             FillEncodedByteSlice(96, 97),
						},
					},
					{
						Proof: FillEncodedByteArraySlice(33, FillEncodedByteSlice(32, 98)),
						Data: &apimiddleware.Deposit_DataJson{
							PublicKey:             FillEncodedByteSlice(48, 100),
							WithdrawalCredentials: FillEncodedByteSlice(32, 101),
							Amount:                "102",
							Signature:             FillEncodedByteSlice(96, 103),
						},
					},
				},
				VoluntaryExits: []*apimiddleware.SignedVoluntaryExitJson{
					{
						Exit: &apimiddleware.VoluntaryExitJson{
							Epoch:          "104",
							ValidatorIndex: "105",
						},
						Signature: FillEncodedByteSlice(96, 106),
					},
					{
						Exit: &apimiddleware.VoluntaryExitJson{
							Epoch:          "107",
							ValidatorIndex: "108",
						},
						Signature: FillEncodedByteSlice(96, 109),
					},
				},
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      FillEncodedByteSlice(64, 110),
					SyncCommitteeSignature: FillEncodedByteSlice(96, 111),
				},
				ExecutionPayloadHeader: &apimiddleware.ExecutionPayloadHeaderDenebJson{
					ParentHash:       FillEncodedByteSlice(32, 112),
					FeeRecipient:     FillEncodedByteSlice(20, 113),
					StateRoot:        FillEncodedByteSlice(32, 114),
					ReceiptsRoot:     FillEncodedByteSlice(32, 115),
					LogsBloom:        FillEncodedByteSlice(256, 116),
					PrevRandao:       FillEncodedByteSlice(32, 117),
					BlockNumber:      "118",
					GasLimit:         "119",
					GasUsed:          "120",
					TimeStamp:        "121",
					ExtraData:        FillEncodedByteSlice(32, 122),
					BaseFeePerGas:    bytesutil.LittleEndianBytesToBigInt(FillByteSlice(32, 123)).String(),
					BlockHash:        FillEncodedByteSlice(32, 124),
					TransactionsRoot: FillEncodedByteSlice(32, 125),
					WithdrawalsRoot:  FillEncodedByteSlice(32, 126),
					BlobGasUsed:      "127",
					ExcessBlobGas:    "128",
				},
				BLSToExecutionChanges: []*apimiddleware.SignedBLSToExecutionChangeJson{
					{
						Message: &apimiddleware.BLSToExecutionChangeJson{
							ValidatorIndex:     "129",
							FromBLSPubkey:      FillEncodedByteSlice(48, 130),
							ToExecutionAddress: FillEncodedByteSlice(20, 131),
						},
						Signature: FillEncodedByteSlice(96, 132),
					},
					{
						Message: &apimiddleware.BLSToExecutionChangeJson{
							ValidatorIndex:     "133",
							FromBLSPubkey:      FillEncodedByteSlice(48, 134),
							ToExecutionAddress: FillEncodedByteSlice(20, 135),
						},
						Signature: FillEncodedByteSlice(96, 136),
					},
				},
				BlobKzgCommitments: []string{FillEncodedByteSlice(48, 137), FillEncodedByteSlice(48, 138)},
			},
		},
		BlindedBlobSidecars: []*apimiddleware.BlindedBlobSidecarJson{
			{
				BlockRoot:       FillEncodedByteSlice(32, 139),
				Index:           "140",
				Slot:            "141",
				BlockParentRoot: FillEncodedByteSlice(32, 142),
				ProposerIndex:   "143",
				BlobRoot:        FillEncodedByteSlice(32, 144),
				KzgCommitment:   FillEncodedByteSlice(48, 145),
				KzgProof:        FillEncodedByteSlice(48, 146),
			},
			{
				BlockRoot:       FillEncodedByteSlice(32, 147),
				Index:           "148",
				Slot:            "149",
				BlockParentRoot: FillEncodedByteSlice(32, 150),
				ProposerIndex:   "151",
				BlobRoot:        FillEncodedByteSlice(32, 152),
				KzgCommitment:   FillEncodedByteSlice(48, 153),
				KzgProof:        FillEncodedByteSlice(48, 154),
			},
		},
	}
}
