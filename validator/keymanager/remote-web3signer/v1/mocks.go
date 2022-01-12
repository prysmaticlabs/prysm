package v1

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

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
