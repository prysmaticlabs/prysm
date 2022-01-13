package v1

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestGetAggregateAndProofSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *AggregateAndProofSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockAggregateAndProofSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAggregateAndProofSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAggregateAndProofSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAggregateAndProofSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAggregationSlotSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *AggregationSlotSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockAggregationSlotSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAggregationSlotSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAggregationSlotSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAggregationSlotSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAttestationSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *AttestationSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: MockAttestationSignRequest(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAttestationSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAttestationSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAttestationSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *BlockSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockBlockSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBlockSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockV2AltairSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *BlockV2AltairSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockBlockV2AltairSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBlockV2AltairSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockV2AltairSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockV2AltairSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRandaoRevealSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *RandaoRevealSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockRandaoRevealSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRandaoRevealSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRandaoRevealSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRandaoRevealSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSyncCommitteeContributionAndProofSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *SyncCommitteeContributionAndProofSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockSyncCommitteeContributionAndProofSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSyncCommitteeContributionAndProofSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSyncCommitteeContributionAndProofSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSyncCommitteeContributionAndProofSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSyncCommitteeMessageSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *SyncCommitteeMessageSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockSyncCommitteeMessageSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSyncCommitteeMessageSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSyncCommitteeMessageSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSyncCommitteeMessageSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSyncCommitteeSelectionProofSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *SyncCommitteeSelectionProofSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockSyncCommitteeSelectionProofSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSyncCommitteeSelectionProofSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSyncCommitteeSelectionProofSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSyncCommitteeSelectionProofSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVoluntaryExitSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *VoluntaryExitSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: &validatorpb.SignRequest{
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
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockVoluntaryExitSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetVoluntaryExitSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVoluntaryExitSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVoluntaryExitSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
