package v1

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/testing/util"
)

/////////////////////////////////////////////////////////////////////////////////////////////////
//////////////// Mock Requests //////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////

// GetMockSignRequest returns a mock SignRequest by type.
func GetMockSignRequest(t string) *validatorpb.SignRequest {
	switch t {
	case "AGGREGATION_SLOT":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_Slot{
				Slot: 0,
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "AGGREGATE_AND_PROOF":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_AggregateAttestationAndProof{
				AggregateAttestationAndProof: &eth.AggregateAttestationAndProof{
					AggregatorIndex: 0,
					Aggregate:       util.NewAttestation(),
					SelectionProof:  make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "ATTESTATION":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_AttestationData{
				AttestationData: util.NewAttestation().Data,
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "BLOCK":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_Block{
				Block: &eth.BeaconBlock{
					Slot:          0,
					ProposerIndex: 0,
					ParentRoot:    make([]byte, fieldparams.RootLength),
					StateRoot:     make([]byte, fieldparams.RootLength),
					Body: &eth.BeaconBlockBody{
						RandaoReveal: make([]byte, 32),
						Eth1Data: &eth.Eth1Data{
							DepositRoot:  make([]byte, fieldparams.RootLength),
							DepositCount: 0,
							BlockHash:    make([]byte, 32),
						},
						Graffiti: make([]byte, 32),
						ProposerSlashings: []*eth.ProposerSlashing{
							{
								Header_1: &eth.SignedBeaconBlockHeader{
									Header: &eth.BeaconBlockHeader{
										Slot:          0,
										ProposerIndex: 0,
										ParentRoot:    make([]byte, fieldparams.RootLength),
										StateRoot:     make([]byte, fieldparams.RootLength),
										BodyRoot:      make([]byte, fieldparams.RootLength),
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
								Header_2: &eth.SignedBeaconBlockHeader{
									Header: &eth.BeaconBlockHeader{
										Slot:          0,
										ProposerIndex: 0,
										ParentRoot:    make([]byte, fieldparams.RootLength),
										StateRoot:     make([]byte, fieldparams.RootLength),
										BodyRoot:      make([]byte, fieldparams.RootLength),
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						AttesterSlashings: []*eth.AttesterSlashing{
							{
								Attestation_1: &eth.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data:             util.NewAttestation().Data,
									Signature:        make([]byte, fieldparams.BLSSignatureLength),
								},
								Attestation_2: &eth.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data:             util.NewAttestation().Data,
									Signature:        make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						Attestations: []*eth.Attestation{
							util.NewAttestation(),
						},
						Deposits: []*eth.Deposit{
							{
								Proof: [][]byte{[]byte("A")},
								Data: &eth.Deposit_Data{
									PublicKey:             make([]byte, fieldparams.BLSPubkeyLength),
									WithdrawalCredentials: make([]byte, 32),
									Amount:                0,
									Signature:             make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						VoluntaryExits: []*eth.SignedVoluntaryExit{
							{
								Exit: &eth.VoluntaryExit{
									Epoch:          0,
									ValidatorIndex: 0,
								},
								Signature: make([]byte, fieldparams.BLSSignatureLength),
							},
						},
					},
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "BLOCK_V2":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_BlockV2{
				BlockV2: &eth.BeaconBlockAltair{
					Slot:          0,
					ProposerIndex: 0,
					ParentRoot:    make([]byte, fieldparams.RootLength),
					StateRoot:     make([]byte, fieldparams.RootLength),
					Body: &eth.BeaconBlockBodyAltair{
						RandaoReveal: make([]byte, 32),
						Eth1Data: &eth.Eth1Data{
							DepositRoot:  make([]byte, fieldparams.RootLength),
							DepositCount: 0,
							BlockHash:    make([]byte, 32),
						},
						Graffiti: make([]byte, 32),
						ProposerSlashings: []*eth.ProposerSlashing{
							{
								Header_1: &eth.SignedBeaconBlockHeader{
									Header: &eth.BeaconBlockHeader{
										Slot:          0,
										ProposerIndex: 0,
										ParentRoot:    make([]byte, fieldparams.RootLength),
										StateRoot:     make([]byte, fieldparams.RootLength),
										BodyRoot:      make([]byte, fieldparams.RootLength),
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
								Header_2: &eth.SignedBeaconBlockHeader{
									Header: &eth.BeaconBlockHeader{
										Slot:          0,
										ProposerIndex: 0,
										ParentRoot:    make([]byte, fieldparams.RootLength),
										StateRoot:     make([]byte, fieldparams.RootLength),
										BodyRoot:      make([]byte, fieldparams.RootLength),
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						AttesterSlashings: []*eth.AttesterSlashing{
							{
								Attestation_1: &eth.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data:             util.NewAttestation().Data,
									Signature:        make([]byte, fieldparams.BLSSignatureLength),
								},
								Attestation_2: &eth.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data:             util.NewAttestation().Data,
									Signature:        make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						Attestations: []*eth.Attestation{
							util.NewAttestation(),
						},
						Deposits: []*eth.Deposit{
							{
								Proof: [][]byte{[]byte("A")},
								Data: &eth.Deposit_Data{
									PublicKey:             make([]byte, fieldparams.BLSPubkeyLength),
									WithdrawalCredentials: make([]byte, 32),
									Amount:                0,
									Signature:             make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						VoluntaryExits: []*eth.SignedVoluntaryExit{
							{
								Exit: &eth.VoluntaryExit{
									Epoch:          0,
									ValidatorIndex: 0,
								},
								Signature: make([]byte, fieldparams.BLSSignatureLength),
							},
						},
						SyncAggregate: &eth.SyncAggregate{
							SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
							SyncCommitteeBits:      bitfield.NewBitvector512(),
						},
					},
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "RANDAO_REVEAL":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_Epoch{
				Epoch: 0,
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_ContributionAndProof{
				ContributionAndProof: &eth.ContributionAndProof{
					AggregatorIndex: 0,
					Contribution: &eth.SyncCommitteeContribution{
						Slot:              0,
						BlockRoot:         make([]byte, fieldparams.RootLength),
						SubcommitteeIndex: 0,
						AggregationBits:   bitfield.NewBitvector128(),
						Signature:         make([]byte, fieldparams.BLSSignatureLength),
					},
					SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "SYNC_COMMITTEE_MESSAGE":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_SyncMessageBlockRoot{
				SyncMessageBlockRoot: &validatorpb.SyncMessageBlockRoot{
					Slot:                 0,
					SyncMessageBlockRoot: make([]byte, fieldparams.RootLength),
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "SYNC_COMMITTEE_SELECTION_PROOF":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_SyncAggregatorSelectionData{
				SyncAggregatorSelectionData: &eth.SyncAggregatorSelectionData{
					Slot:              0,
					SubcommitteeIndex: 0,
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	case "VOLUNTARY_EXIT":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_Exit{
				Exit: &eth.VoluntaryExit{
					Epoch:          0,
					ValidatorIndex: 0,
				},
			},
			Fork: &eth.Fork{
				PreviousVersion: make([]byte, 4),
				CurrentVersion:  make([]byte, 4),
				Epoch:           0,
			},
		}
	default:
		fmt.Printf("Web3signer sign request type: %v  not found", t)
		return nil
	}
}

// MockAggregationSlotSignRequest is a mock implementation of the AggregationSlotSignRequest.
func MockAggregationSlotSignRequest() *AggregationSlotSignRequest {
	return &AggregationSlotSignRequest{
		Type:            "AGGREGATION_SLOT",
		ForkInfo:        MockForkInfo(),
		SigningRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
		AggregationSlot: &AggregationSlot{Slot: "0"},
	}
}

// MockAggregateAndProofSignRequest is a mock implementation of the AggregateAndProofSignRequest.
func MockAggregateAndProofSignRequest() *AggregateAndProofSignRequest {
	return &AggregateAndProofSignRequest{
		Type:        "AGGREGATE_AND_PROOF",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		AggregateAndProof: &AggregateAndProof{
			AggregatorIndex: "0",
			Aggregate:       MockAttestation(),
			SelectionProof:  hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
		},
	}
}

// MockAttestationSignRequest is a mock implementation of the AttestationSignRequest.
func MockAttestationSignRequest() *AttestationSignRequest {
	return &AttestationSignRequest{
		Type:        "ATTESTATION",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		Attestation: MockAttestation().Data,
	}
}

// MockBlockSignRequest is a mock implementation of the BlockSignRequest.
func MockBlockSignRequest() *BlockSignRequest {
	return &BlockSignRequest{
		Type:        "BLOCK",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		Block: &BeaconBlock{
			Slot:          "0",
			ProposerIndex: "0",
			ParentRoot:    hexutil.Encode(make([]byte, fieldparams.RootLength)),
			StateRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
			Body:          MockBeaconBlockBody(),
		},
	}
}

// MockBlockV2AltairSignRequest is a mock implementation of the BlockV2AltairSignRequest.
func MockBlockV2AltairSignRequest() *BlockV2AltairSignRequest {
	return &BlockV2AltairSignRequest{
		Type:        "BLOCK_V2",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		BeaconBlock: &BeaconBlockAltairBlockV2{
			Version: "ALTAIR",
			Block:   MockBeaconBlockAltair(),
		},
	}
}

// MockRandaoRevealSignRequest is a mock implementation of the RandaoRevealSignRequest.
func MockRandaoRevealSignRequest() *RandaoRevealSignRequest {
	return &RandaoRevealSignRequest{
		Type:        "RANDAO_REVEAL",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		RandaoReveal: &RandaoReveal{
			Epoch: "0",
		},
	}
}

// MockSyncCommitteeContributionAndProofSignRequest is a mock implementation of the SyncCommitteeContributionAndProofSignRequest.
func MockSyncCommitteeContributionAndProofSignRequest() *SyncCommitteeContributionAndProofSignRequest {
	return &SyncCommitteeContributionAndProofSignRequest{
		Type:                 "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF",
		ForkInfo:             MockForkInfo(),
		SigningRoot:          hexutil.Encode(make([]byte, fieldparams.RootLength)),
		ContributionAndProof: MockContributionAndProof(),
	}
}

// MockSyncCommitteeMessageSignRequest is a mock implementation of the SyncCommitteeMessageSignRequest.
func MockSyncCommitteeMessageSignRequest() *SyncCommitteeMessageSignRequest {
	return &SyncCommitteeMessageSignRequest{
		Type:        "SYNC_COMMITTEE_MESSAGE",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		SyncCommitteeMessage: &SyncCommitteeMessage{
			BeaconBlockRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
			Slot:            "0",
		},
	}
}

// MockSyncCommitteeSelectionProofSignRequest is a mock implementation of the SyncCommitteeSelectionProofSignRequest.
func MockSyncCommitteeSelectionProofSignRequest() *SyncCommitteeSelectionProofSignRequest {
	return &SyncCommitteeSelectionProofSignRequest{
		Type:        "SYNC_COMMITTEE_SELECTION_PROOF",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		SyncAggregatorSelectionData: &SyncAggregatorSelectionData{
			Slot:              "0",
			SubcommitteeIndex: "0",
		},
	}
}

// MockVoluntaryExitSignRequest is a mock implementation of the VoluntaryExitSignRequest.
func MockVoluntaryExitSignRequest() *VoluntaryExitSignRequest {
	return &VoluntaryExitSignRequest{
		Type:        "VOLUNTARY_EXIT",
		ForkInfo:    MockForkInfo(),
		SigningRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
		VoluntaryExit: &VoluntaryExit{
			Epoch:          "0",
			ValidatorIndex: "0",
		},
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////

// MockForkInfo is a mock implementation of the ForkInfo.
func MockForkInfo() *ForkInfo {
	return &ForkInfo{
		Fork: &Fork{
			PreviousVersion: hexutil.Encode(make([]byte, 4)),
			CurrentVersion:  hexutil.Encode(make([]byte, 4)),
			Epoch:           "0",
		},
		GenesisValidatorsRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
	}

}

// MockAttestation is a mock implementation of the Attestation.
func MockAttestation() *Attestation {
	return &Attestation{
		AggregationBits: hexutil.Encode(bitfield.Bitlist{0b1101}.Bytes()),
		Data: &AttestationData{
			Slot:            "0",
			Index:           "0",
			BeaconBlockRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
			Source: &Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
			Target: &Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
		},
		Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
	}
}

func MockIndexedAttestation() *IndexedAttestation {
	return &IndexedAttestation{
		AttestingIndices: []string{"0", "1", "2"},
		Data: &AttestationData{
			Slot:            "0",
			Index:           "0",
			BeaconBlockRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
			Source: &Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
			Target: &Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
		},
		Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
	}
}

func MockBeaconBlockAltair() *BeaconBlockAltair {
	return &BeaconBlockAltair{
		Slot:          "0",
		ProposerIndex: "0",
		ParentRoot:    hexutil.Encode(make([]byte, fieldparams.RootLength)),
		StateRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
		Body: &BeaconBlockBodyAltair{
			RandaoReveal: hexutil.Encode(make([]byte, 32)),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
				DepositCount: "0",
				BlockHash:    hexutil.Encode(make([]byte, 32)),
			},
			Graffiti: hexutil.Encode(make([]byte, 32)),
			ProposerSlashings: []*ProposerSlashing{
				{
					SignedHeader_1: &SignedBeaconBlockHeader{
						Message: &BeaconBlockHeader{
							Slot:          "0",
							ProposerIndex: "0",
							ParentRoot:    hexutil.Encode(make([]byte, fieldparams.RootLength)),
							StateRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
							BodyRoot:      hexutil.Encode(make([]byte, fieldparams.RootLength)),
						},
						Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
					},
					SignedHeader_2: &SignedBeaconBlockHeader{
						Message: &BeaconBlockHeader{
							Slot:          "0",
							ProposerIndex: "0",
							ParentRoot:    hexutil.Encode(make([]byte, fieldparams.RootLength)),
							StateRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
							BodyRoot:      hexutil.Encode(make([]byte, fieldparams.RootLength)),
						},
						Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
					},
				},
			},
			AttesterSlashings: []*AttesterSlashing{
				{
					Attestation_1: MockIndexedAttestation(),
					Attestation_2: MockIndexedAttestation(),
				},
			},
			Attestations: []*Attestation{
				MockAttestation(),
			},
			Deposits: []*Deposit{
				{
					Proof: []string{"0x41"},
					Data: &DepositData{
						PublicKey:             hexutil.Encode(make([]byte, fieldparams.BLSPubkeyLength)),
						WithdrawalCredentials: hexutil.Encode(make([]byte, 32)),
						Amount:                "0",
						Signature:             hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
					},
				},
			},
			VoluntaryExits: []*SignedVoluntaryExit{
				{
					Message: &VoluntaryExit{
						Epoch:          "0",
						ValidatorIndex: "0",
					},
					Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
				},
			},
			SyncAggregate: &SyncAggregate{
				SyncCommitteeSignature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
				SyncCommitteeBits:      hexutil.Encode(bitfield.NewBitvector512().Bytes()),
			},
		},
	}
}

func MockBeaconBlockBody() *BeaconBlockBody {
	return &BeaconBlockBody{
		RandaoReveal: hexutil.Encode(make([]byte, 32)),
		Eth1Data: &Eth1Data{
			DepositRoot:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			DepositCount: "0",
			BlockHash:    hexutil.Encode(make([]byte, 32)),
		},
		Graffiti: hexutil.Encode(make([]byte, 32)),
		ProposerSlashings: []*ProposerSlashing{
			{
				SignedHeader_1: &SignedBeaconBlockHeader{
					Message: &BeaconBlockHeader{
						Slot:          "0",
						ProposerIndex: "0",
						ParentRoot:    hexutil.Encode(make([]byte, fieldparams.RootLength)),
						StateRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
						BodyRoot:      hexutil.Encode(make([]byte, fieldparams.RootLength)),
					},
					Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
				},
				SignedHeader_2: &SignedBeaconBlockHeader{
					Message: &BeaconBlockHeader{
						Slot:          "0",
						ProposerIndex: "0",
						ParentRoot:    hexutil.Encode(make([]byte, fieldparams.RootLength)),
						StateRoot:     hexutil.Encode(make([]byte, fieldparams.RootLength)),
						BodyRoot:      hexutil.Encode(make([]byte, fieldparams.RootLength)),
					},
					Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
				},
			},
		},
		AttesterSlashings: []*AttesterSlashing{
			{
				Attestation_1: MockIndexedAttestation(),
				Attestation_2: MockIndexedAttestation(),
			},
		},
		Attestations: []*Attestation{
			MockAttestation(),
		},
		Deposits: []*Deposit{
			{
				Proof: []string{"0x41"},
				Data: &DepositData{
					PublicKey:             hexutil.Encode(make([]byte, fieldparams.BLSPubkeyLength)),
					WithdrawalCredentials: hexutil.Encode(make([]byte, 32)),
					Amount:                "0",
					Signature:             hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
				},
			},
		},
		VoluntaryExits: []*SignedVoluntaryExit{
			{
				Message: &VoluntaryExit{
					Epoch:          "0",
					ValidatorIndex: "0",
				},
				Signature: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
			},
		},
	}
}

func MockContributionAndProof() *ContributionAndProof {
	return &ContributionAndProof{
		AggregatorIndex: "0",
		Contribution: &SyncCommitteeContribution{
			Slot:              "0",
			BeaconBlockRoot:   hexutil.Encode(make([]byte, fieldparams.RootLength)),
			SubcommitteeIndex: "0",
			AggregationBits:   hexutil.Encode(bitfield.NewBitvector128().Bytes()),
			Signature:         hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
		},
		SelectionProof: hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
	}
}
