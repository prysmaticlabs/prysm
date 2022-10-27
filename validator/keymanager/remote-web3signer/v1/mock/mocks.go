package mock

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	v1 "github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer/v1"
)

/////////////////////////////////////////////////////////////////////////////////////////////////
//////////////// Mock Requests //////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////

func MockSyncComitteeBits() []byte {
	currSize := new(eth.SyncAggregate).SyncCommitteeBits.Len()
	switch currSize {
	case 512:
		return bitfield.NewBitvector512()
	case 32:
		return bitfield.NewBitvector32()
	default:
		return nil
	}
}

func MockAggregationBits() []byte {
	currSize := new(eth.SyncCommitteeContribution).AggregationBits.Len()
	switch currSize {
	case 128:
		return bitfield.NewBitvector128()
	case 8:
		return bitfield.NewBitvector8()
	default:
		return nil
	}
}

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
			SigningSlot: 0,
		}
	case "AGGREGATE_AND_PROOF":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_AggregateAttestationAndProof{
				AggregateAttestationAndProof: &eth.AggregateAttestationAndProof{
					AggregatorIndex: 0,
					Aggregate: &eth.Attestation{
						AggregationBits: bitfield.Bitlist{0b1101},
						Data: &eth.AttestationData{
							BeaconBlockRoot: make([]byte, fieldparams.RootLength),
							Source: &eth.Checkpoint{
								Root: make([]byte, fieldparams.RootLength),
							},
							Target: &eth.Checkpoint{
								Root: make([]byte, fieldparams.RootLength),
							},
						},
						Signature: make([]byte, 96),
					},
					SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			SigningSlot: 0,
		}
	case "ATTESTATION":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_AttestationData{
				AttestationData: &eth.AttestationData{
					BeaconBlockRoot: make([]byte, fieldparams.RootLength),
					Source: &eth.Checkpoint{
						Root: make([]byte, fieldparams.RootLength),
					},
					Target: &eth.Checkpoint{
						Root: make([]byte, fieldparams.RootLength),
					},
				},
			},
			SigningSlot: 0,
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
									Data: &eth.AttestationData{
										BeaconBlockRoot: make([]byte, fieldparams.RootLength),
										Source: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
										Target: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
								Attestation_2: &eth.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data: &eth.AttestationData{
										BeaconBlockRoot: make([]byte, fieldparams.RootLength),
										Source: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
										Target: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						Attestations: []*eth.Attestation{
							{
								AggregationBits: bitfield.Bitlist{0b1101},
								Data: &eth.AttestationData{
									BeaconBlockRoot: make([]byte, fieldparams.RootLength),
									Source: &eth.Checkpoint{
										Root: make([]byte, fieldparams.RootLength),
									},
									Target: &eth.Checkpoint{
										Root: make([]byte, fieldparams.RootLength),
									},
								},
								Signature: make([]byte, 96),
							},
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
			SigningSlot: 0,
		}
	case "BLOCK_V2", "BLOCK_V2_ALTAIR":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_BlockAltair{
				BlockAltair: &eth.BeaconBlockAltair{
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
									Data: &eth.AttestationData{
										BeaconBlockRoot: make([]byte, fieldparams.RootLength),
										Source: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
										Target: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
								Attestation_2: &eth.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data: &eth.AttestationData{
										BeaconBlockRoot: make([]byte, fieldparams.RootLength),
										Source: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
										Target: &eth.Checkpoint{
											Root: make([]byte, fieldparams.RootLength),
										},
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						Attestations: []*eth.Attestation{
							{
								AggregationBits: bitfield.Bitlist{0b1101},
								Data: &eth.AttestationData{
									BeaconBlockRoot: make([]byte, fieldparams.RootLength),
									Source: &eth.Checkpoint{
										Root: make([]byte, fieldparams.RootLength),
									},
									Target: &eth.Checkpoint{
										Root: make([]byte, fieldparams.RootLength),
									},
								},
								Signature: make([]byte, 96),
							},
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
							SyncCommitteeBits:      MockSyncComitteeBits(),
						},
					},
				},
			},
			SigningSlot: 0,
		}
	case "BLOCK_V2_BELLATRIX":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_BlockBellatrix{
				BlockBellatrix: util.HydrateBeaconBlockBellatrix(&eth.BeaconBlockBellatrix{}),
			},
		}
	case "BLOCK_V2_BLINDED_BELLATRIX":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_BlindedBlockBellatrix{
				BlindedBlockBellatrix: util.HydrateBlindedBeaconBlockBellatrix(&eth.BlindedBeaconBlockBellatrix{}),
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
			SigningSlot: 0,
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
						AggregationBits:   MockAggregationBits(),
						Signature:         make([]byte, fieldparams.BLSSignatureLength),
					},
					SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			SigningSlot: 0,
		}
	case "SYNC_COMMITTEE_MESSAGE":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_SyncMessageBlockRoot{
				SyncMessageBlockRoot: make([]byte, fieldparams.RootLength),
			},
			SigningSlot: 0,
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
			SigningSlot: 0,
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
			SigningSlot: 0,
		}
	case "VALIDATOR_REGISTRATION":
		return &validatorpb.SignRequest{
			PublicKey:       make([]byte, fieldparams.BLSPubkeyLength),
			SigningRoot:     make([]byte, fieldparams.RootLength),
			SignatureDomain: make([]byte, 4),
			Object: &validatorpb.SignRequest_Registration{
				Registration: &eth.ValidatorRegistrationV1{
					FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
					GasLimit:     uint64(0),
					Timestamp:    uint64(0),
					Pubkey:       make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			SigningSlot: 0,
		}
	default:
		fmt.Printf("Web3signer sign request type: %v  not found", t)
		return nil
	}
}

// MockAggregationSlotSignRequest is a mock implementation of the AggregationSlotSignRequest.
func MockAggregationSlotSignRequest() *v1.AggregationSlotSignRequest {
	return &v1.AggregationSlotSignRequest{
		Type:            "AGGREGATION_SLOT",
		ForkInfo:        MockForkInfo(),
		SigningRoot:     make([]byte, fieldparams.RootLength),
		AggregationSlot: &v1.AggregationSlot{Slot: "0"},
	}
}

// MockAggregateAndProofSignRequest is a mock implementation of the AggregateAndProofSignRequest.
func MockAggregateAndProofSignRequest() *v1.AggregateAndProofSignRequest {
	return &v1.AggregateAndProofSignRequest{
		Type:        "AGGREGATE_AND_PROOF",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		AggregateAndProof: &v1.AggregateAndProof{
			AggregatorIndex: "0",
			Aggregate:       MockAttestation(),
			SelectionProof:  make([]byte, fieldparams.BLSSignatureLength),
		},
	}
}

// MockAttestationSignRequest is a mock implementation of the AttestationSignRequest.
func MockAttestationSignRequest() *v1.AttestationSignRequest {
	return &v1.AttestationSignRequest{
		Type:        "ATTESTATION",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		Attestation: MockAttestation().Data,
	}
}

// MockBlockSignRequest is a mock implementation of the BlockSignRequest.
func MockBlockSignRequest() *v1.BlockSignRequest {
	return &v1.BlockSignRequest{
		Type:        "BLOCK",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		Block: &v1.BeaconBlock{
			Slot:          "0",
			ProposerIndex: "0",
			ParentRoot:    make([]byte, fieldparams.RootLength),
			StateRoot:     make([]byte, fieldparams.RootLength),
			Body:          MockBeaconBlockBody(),
		},
	}
}

// MockBlockV2AltairSignRequest is a mock implementation of the BlockAltairSignRequest.
func MockBlockV2AltairSignRequest() *v1.BlockAltairSignRequest {
	return &v1.BlockAltairSignRequest{
		Type:        "BLOCK_V2",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		BeaconBlock: &v1.BeaconBlockAltairBlockV2{
			Version: "ALTAIR",
			Block:   MockBeaconBlockAltair(),
		},
	}
}

func MockBlockV2BellatrixSignRequest(bodyRoot []byte) *v1.BlockBellatrixSignRequest {
	return &v1.BlockBellatrixSignRequest{
		Type:        "BLOCK_V2",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		BeaconBlock: &v1.BeaconBlockBellatrixBlockV2{
			Version: "BELLATRIX",
			BlockHeader: &v1.BeaconBlockHeader{
				Slot:          "0",
				ProposerIndex: "0",
				ParentRoot:    make([]byte, fieldparams.RootLength),
				StateRoot:     make([]byte, fieldparams.RootLength),
				BodyRoot:      bodyRoot,
			},
		},
	}
}

// MockRandaoRevealSignRequest is a mock implementation of the RandaoRevealSignRequest.
func MockRandaoRevealSignRequest() *v1.RandaoRevealSignRequest {
	return &v1.RandaoRevealSignRequest{
		Type:        "RANDAO_REVEAL",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		RandaoReveal: &v1.RandaoReveal{
			Epoch: "0",
		},
	}
}

// MockSyncCommitteeContributionAndProofSignRequest is a mock implementation of the SyncCommitteeContributionAndProofSignRequest.
func MockSyncCommitteeContributionAndProofSignRequest() *v1.SyncCommitteeContributionAndProofSignRequest {
	return &v1.SyncCommitteeContributionAndProofSignRequest{
		Type:                 "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF",
		ForkInfo:             MockForkInfo(),
		SigningRoot:          make([]byte, fieldparams.RootLength),
		ContributionAndProof: MockContributionAndProof(),
	}
}

// MockSyncCommitteeMessageSignRequest is a mock implementation of the SyncCommitteeMessageSignRequest.
func MockSyncCommitteeMessageSignRequest() *v1.SyncCommitteeMessageSignRequest {
	return &v1.SyncCommitteeMessageSignRequest{
		Type:        "SYNC_COMMITTEE_MESSAGE",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		SyncCommitteeMessage: &v1.SyncCommitteeMessage{
			BeaconBlockRoot: make([]byte, fieldparams.RootLength),
			Slot:            "0",
		},
	}
}

// MockSyncCommitteeSelectionProofSignRequest is a mock implementation of the SyncCommitteeSelectionProofSignRequest.
func MockSyncCommitteeSelectionProofSignRequest() *v1.SyncCommitteeSelectionProofSignRequest {
	return &v1.SyncCommitteeSelectionProofSignRequest{
		Type:        "SYNC_COMMITTEE_SELECTION_PROOF",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		SyncAggregatorSelectionData: &v1.SyncAggregatorSelectionData{
			Slot:              "0",
			SubcommitteeIndex: "0",
		},
	}
}

// MockVoluntaryExitSignRequest is a mock implementation of the VoluntaryExitSignRequest.
func MockVoluntaryExitSignRequest() *v1.VoluntaryExitSignRequest {
	return &v1.VoluntaryExitSignRequest{
		Type:        "VOLUNTARY_EXIT",
		ForkInfo:    MockForkInfo(),
		SigningRoot: make([]byte, fieldparams.RootLength),
		VoluntaryExit: &v1.VoluntaryExit{
			Epoch:          "0",
			ValidatorIndex: "0",
		},
	}
}

// MockValidatorRegistrationSignRequest is a mock implementation of the ValidatorRegistrationSignRequest.
func MockValidatorRegistrationSignRequest() *v1.ValidatorRegistrationSignRequest {
	return &v1.ValidatorRegistrationSignRequest{
		Type:        "VALIDATOR_REGISTRATION",
		SigningRoot: make([]byte, fieldparams.RootLength),
		ValidatorRegistration: &v1.ValidatorRegistration{
			FeeRecipient: make([]byte, fieldparams.FeeRecipientLength),
			GasLimit:     fmt.Sprint(0),
			Timestamp:    fmt.Sprint(0),
			Pubkey:       make([]byte, fieldparams.BLSSignatureLength),
		},
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////////////

// MockForkInfo is a mock implementation of the ForkInfo.
func MockForkInfo() *v1.ForkInfo {
	return &v1.ForkInfo{
		Fork: &v1.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
			Epoch:           "0",
		},
		GenesisValidatorsRoot: make([]byte, fieldparams.RootLength),
	}

}

// MockAttestation is a mock implementation of the Attestation.
func MockAttestation() *v1.Attestation {
	return &v1.Attestation{
		AggregationBits: []byte(bitfield.Bitlist{0b1101}),
		Data: &v1.AttestationData{
			Slot:            "0",
			Index:           "0",
			BeaconBlockRoot: make([]byte, fieldparams.RootLength),
			Source: &v1.Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
			Target: &v1.Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
		},
		Signature: make([]byte, fieldparams.BLSSignatureLength),
	}
}

func MockIndexedAttestation() *v1.IndexedAttestation {
	return &v1.IndexedAttestation{
		AttestingIndices: []string{"0", "1", "2"},
		Data: &v1.AttestationData{
			Slot:            "0",
			Index:           "0",
			BeaconBlockRoot: make([]byte, fieldparams.RootLength),
			Source: &v1.Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
			Target: &v1.Checkpoint{
				Epoch: "0",
				Root:  hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
		},
		Signature: make([]byte, fieldparams.BLSSignatureLength),
	}
}

func MockBeaconBlockAltair() *v1.BeaconBlockAltair {
	return &v1.BeaconBlockAltair{
		Slot:          "0",
		ProposerIndex: "0",
		ParentRoot:    make([]byte, fieldparams.RootLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		Body: &v1.BeaconBlockBodyAltair{
			RandaoReveal: make([]byte, 32),
			Eth1Data: &v1.Eth1Data{
				DepositRoot:  make([]byte, fieldparams.RootLength),
				DepositCount: "0",
				BlockHash:    make([]byte, 32),
			},
			Graffiti: make([]byte, 32),
			ProposerSlashings: []*v1.ProposerSlashing{
				{
					Signedheader1: &v1.SignedBeaconBlockHeader{
						Message: &v1.BeaconBlockHeader{
							Slot:          "0",
							ProposerIndex: "0",
							ParentRoot:    make([]byte, fieldparams.RootLength),
							StateRoot:     make([]byte, fieldparams.RootLength),
							BodyRoot:      make([]byte, fieldparams.RootLength),
						},
						Signature: make([]byte, fieldparams.BLSSignatureLength),
					},
					Signedheader2: &v1.SignedBeaconBlockHeader{
						Message: &v1.BeaconBlockHeader{
							Slot:          "0",
							ProposerIndex: "0",
							ParentRoot:    make([]byte, fieldparams.RootLength),
							StateRoot:     make([]byte, fieldparams.RootLength),
							BodyRoot:      make([]byte, fieldparams.RootLength),
						},
						Signature: make([]byte, fieldparams.BLSSignatureLength),
					},
				},
			},
			AttesterSlashings: []*v1.AttesterSlashing{
				{
					Attestation1: MockIndexedAttestation(),
					Attestation2: MockIndexedAttestation(),
				},
			},
			Attestations: []*v1.Attestation{
				MockAttestation(),
			},
			Deposits: []*v1.Deposit{
				{
					Proof: []string{"0x41"},
					Data: &v1.DepositData{
						PublicKey:             make([]byte, fieldparams.BLSPubkeyLength),
						WithdrawalCredentials: make([]byte, 32),
						Amount:                "0",
						Signature:             make([]byte, fieldparams.BLSSignatureLength),
					},
				},
			},
			VoluntaryExits: []*v1.SignedVoluntaryExit{
				{
					Message: &v1.VoluntaryExit{
						Epoch:          "0",
						ValidatorIndex: "0",
					},
					Signature: make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			SyncAggregate: &v1.SyncAggregate{
				SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
				SyncCommitteeBits:      MockSyncComitteeBits(),
			},
		},
	}
}

func MockBeaconBlockBody() *v1.BeaconBlockBody {
	return &v1.BeaconBlockBody{
		RandaoReveal: make([]byte, 32),
		Eth1Data: &v1.Eth1Data{
			DepositRoot:  make([]byte, fieldparams.RootLength),
			DepositCount: "0",
			BlockHash:    make([]byte, 32),
		},
		Graffiti: make([]byte, 32),
		ProposerSlashings: []*v1.ProposerSlashing{
			{
				Signedheader1: &v1.SignedBeaconBlockHeader{
					Message: &v1.BeaconBlockHeader{
						Slot:          "0",
						ProposerIndex: "0",
						ParentRoot:    make([]byte, fieldparams.RootLength),
						StateRoot:     make([]byte, fieldparams.RootLength),
						BodyRoot:      make([]byte, fieldparams.RootLength),
					},
					Signature: make([]byte, fieldparams.BLSSignatureLength),
				},
				Signedheader2: &v1.SignedBeaconBlockHeader{
					Message: &v1.BeaconBlockHeader{
						Slot:          "0",
						ProposerIndex: "0",
						ParentRoot:    make([]byte, fieldparams.RootLength),
						StateRoot:     make([]byte, fieldparams.RootLength),
						BodyRoot:      make([]byte, fieldparams.RootLength),
					},
					Signature: make([]byte, fieldparams.BLSSignatureLength),
				},
			},
		},
		AttesterSlashings: []*v1.AttesterSlashing{
			{
				Attestation1: MockIndexedAttestation(),
				Attestation2: MockIndexedAttestation(),
			},
		},
		Attestations: []*v1.Attestation{
			MockAttestation(),
		},
		Deposits: []*v1.Deposit{
			{
				Proof: []string{"0x41"},
				Data: &v1.DepositData{
					PublicKey:             make([]byte, fieldparams.BLSPubkeyLength),
					WithdrawalCredentials: make([]byte, 32),
					Amount:                "0",
					Signature:             make([]byte, fieldparams.BLSSignatureLength),
				},
			},
		},
		VoluntaryExits: []*v1.SignedVoluntaryExit{
			{
				Message: &v1.VoluntaryExit{
					Epoch:          "0",
					ValidatorIndex: "0",
				},
				Signature: make([]byte, fieldparams.BLSSignatureLength),
			},
		},
	}
}

func MockContributionAndProof() *v1.ContributionAndProof {
	return &v1.ContributionAndProof{
		AggregatorIndex: "0",
		Contribution: &v1.SyncCommitteeContribution{
			Slot:              "0",
			BeaconBlockRoot:   make([]byte, fieldparams.RootLength),
			SubcommitteeIndex: "0",
			AggregationBits:   MockAggregationBits(),
			Signature:         make([]byte, fieldparams.BLSSignatureLength),
		},
		SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
	}
}
