package v1

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/testing/util"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func TestMapAggregateAndProof(t *testing.T) {
	type args struct {
		from *ethpb.AggregateAttestationAndProof
	}
	tests := []struct {
		name    string
		args    args
		want    *AggregateAndProof
		wantErr bool
	}{
		{
			name: "HappyPathTest",
			args: args{
				from: &ethpb.AggregateAttestationAndProof{
					AggregatorIndex: 0,
					Aggregate:       util.NewAttestation(),
					SelectionProof:  make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			want: &AggregateAndProof{
				AggregatorIndex: "0",
				Aggregate:       MockAttestation(),
				SelectionProof:  hexutil.Encode(make([]byte, fieldparams.BLSSignatureLength)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapAggregateAndProof(tt.args.from)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAggregateAndProof() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.Aggregate, tt.want.Aggregate) {
				t.Errorf("MapAggregateAndProof() got = %v, want %v", got.Aggregate, tt.want.Aggregate)
			}
		})
	}
}

func TestMapAttestation(t *testing.T) {
	type args struct {
		attestation *ethpb.Attestation
	}
	tests := []struct {
		name    string
		args    args
		want    *Attestation
		wantErr bool
	}{
		{
			name: "HappyPathTest",
			args: args{
				attestation: util.NewAttestation(),
			},
			want:    MockAttestation(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapAttestation(tt.args.attestation)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAttestation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapAttestation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapAttestationData(t *testing.T) {
	type args struct {
		data *ethpb.AttestationData
	}
	tests := []struct {
		name    string
		args    args
		want    *AttestationData
		wantErr bool
	}{
		{
			name: "HappyPathTest",
			args: args{
				data: util.NewAttestation().Data,
			},
			want:    MockAttestation().Data,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapAttestationData(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAttestationData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapAttestationData() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapAttesterSlashing(t *testing.T) {
	type args struct {
		slashing *ethpb.AttesterSlashing
	}
	tests := []struct {
		name    string
		args    args
		want    *AttesterSlashing
		wantErr bool
	}{
		{
			name: "HappyPathTest",
			args: args{
				slashing: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{0, 1, 2},
						Data:             util.NewAttestation().Data,
						Signature:        make([]byte, fieldparams.BLSSignatureLength),
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{0, 1, 2},
						Data:             util.NewAttestation().Data,
						Signature:        make([]byte, fieldparams.BLSSignatureLength),
					},
				},
			},
			want: &AttesterSlashing{
				Attestation_1: MockIndexedAttestation(),
				Attestation_2: MockIndexedAttestation(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapAttesterSlashing(tt.args.slashing)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAttesterSlashing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.Attestation_1, tt.want.Attestation_1) {
				t.Errorf("MapAttesterSlashing() got = %v, want %v", got.Attestation_1, tt.want.Attestation_1)
			}
		})
	}
}

func TestMapBeaconBlockAltair(t *testing.T) {
	type args struct {
		block *ethpb.BeaconBlockAltair
	}
	tests := []struct {
		name    string
		args    args
		want    *BeaconBlockAltair
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				block: &ethpb.BeaconBlockAltair{
					Slot:          0,
					ProposerIndex: 0,
					ParentRoot:    make([]byte, fieldparams.RootLength),
					StateRoot:     make([]byte, fieldparams.RootLength),
					Body: &ethpb.BeaconBlockBodyAltair{
						RandaoReveal: make([]byte, 32),
						Eth1Data: &ethpb.Eth1Data{
							DepositRoot:  make([]byte, fieldparams.RootLength),
							DepositCount: 0,
							BlockHash:    make([]byte, 32),
						},
						Graffiti: make([]byte, 32),
						ProposerSlashings: []*ethpb.ProposerSlashing{
							{
								Header_1: &ethpb.SignedBeaconBlockHeader{
									Header: &ethpb.BeaconBlockHeader{
										Slot:          0,
										ProposerIndex: 0,
										ParentRoot:    make([]byte, fieldparams.RootLength),
										StateRoot:     make([]byte, fieldparams.RootLength),
										BodyRoot:      make([]byte, fieldparams.RootLength),
									},
									Signature: make([]byte, fieldparams.BLSSignatureLength),
								},
								Header_2: &ethpb.SignedBeaconBlockHeader{
									Header: &ethpb.BeaconBlockHeader{
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
						AttesterSlashings: []*ethpb.AttesterSlashing{
							{
								Attestation_1: &ethpb.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data:             util.NewAttestation().Data,
									Signature:        make([]byte, fieldparams.BLSSignatureLength),
								},
								Attestation_2: &ethpb.IndexedAttestation{
									AttestingIndices: []uint64{0, 1, 2},
									Data:             util.NewAttestation().Data,
									Signature:        make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						Attestations: []*ethpb.Attestation{
							util.NewAttestation(),
						},
						Deposits: []*ethpb.Deposit{
							{
								Proof: [][]byte{[]byte("A")},
								Data: &ethpb.Deposit_Data{
									PublicKey:             make([]byte, fieldparams.BLSPubkeyLength),
									WithdrawalCredentials: make([]byte, 32),
									Amount:                0,
									Signature:             make([]byte, fieldparams.BLSSignatureLength),
								},
							},
						},
						VoluntaryExits: []*ethpb.SignedVoluntaryExit{
							{
								Exit: &ethpb.VoluntaryExit{
									Epoch:          0,
									ValidatorIndex: 0,
								},
								Signature: make([]byte, fieldparams.BLSSignatureLength),
							},
						},
						SyncAggregate: &ethpb.SyncAggregate{
							SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
							SyncCommitteeBits:      make([]byte, 64),
						},
					},
				},
			},
			want:    MockBeaconBlockAltair(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapBeaconBlockAltair(tt.args.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapBeaconBlockAltair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.Body, tt.want.Body) {
				t.Errorf("MapBeaconBlockAltair() got = %v, want %v", got.Body.SyncAggregate, tt.want.Body.SyncAggregate)
			}
		})
	}
}

func TestMapBeaconBlockBody(t *testing.T) {
	type args struct {
		body *ethpb.BeaconBlockBody
	}
	tests := []struct {
		name    string
		args    args
		want    *BeaconBlockBody
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				body: &ethpb.BeaconBlockBody{
					RandaoReveal: make([]byte, 32),
					Eth1Data: &ethpb.Eth1Data{
						DepositRoot:  make([]byte, fieldparams.RootLength),
						DepositCount: 0,
						BlockHash:    make([]byte, 32),
					},
					Graffiti: make([]byte, 32),
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									Slot:          0,
									ProposerIndex: 0,
									ParentRoot:    make([]byte, fieldparams.RootLength),
									StateRoot:     make([]byte, fieldparams.RootLength),
									BodyRoot:      make([]byte, fieldparams.RootLength),
								},
								Signature: make([]byte, fieldparams.BLSSignatureLength),
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
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
					AttesterSlashings: []*ethpb.AttesterSlashing{
						{
							Attestation_1: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{0, 1, 2},
								Data:             util.NewAttestation().Data,
								Signature:        make([]byte, fieldparams.BLSSignatureLength),
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{0, 1, 2},
								Data:             util.NewAttestation().Data,
								Signature:        make([]byte, fieldparams.BLSSignatureLength),
							},
						},
					},
					Attestations: []*ethpb.Attestation{
						util.NewAttestation(),
					},
					Deposits: []*ethpb.Deposit{
						{
							Proof: [][]byte{[]byte("A")},
							Data: &ethpb.Deposit_Data{
								PublicKey:             make([]byte, fieldparams.BLSPubkeyLength),
								WithdrawalCredentials: make([]byte, 32),
								Amount:                0,
								Signature:             make([]byte, fieldparams.BLSSignatureLength),
							},
						},
					},
					VoluntaryExits: []*ethpb.SignedVoluntaryExit{
						{
							Exit: &ethpb.VoluntaryExit{
								Epoch:          0,
								ValidatorIndex: 0,
							},
							Signature: make([]byte, fieldparams.BLSSignatureLength),
						},
					},
				},
			},
			want:    MockBeaconBlockBody(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapBeaconBlockBody(tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapBeaconBlockBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapBeaconBlockBody() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapContributionAndProof(t *testing.T) {
	type args struct {
		contribution *ethpb.ContributionAndProof
	}
	tests := []struct {
		name    string
		args    args
		want    *ContributionAndProof
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				contribution: &ethpb.ContributionAndProof{
					AggregatorIndex: 0,
					Contribution: &ethpb.SyncCommitteeContribution{
						Slot:              0,
						BlockRoot:         make([]byte, fieldparams.RootLength),
						SubcommitteeIndex: 0,
						AggregationBits:   make([]byte, 64),
						Signature:         make([]byte, fieldparams.BLSSignatureLength),
					},
					SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			want: MockContributionAndProof(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapContributionAndProof(tt.args.contribution)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapContributionAndProof() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapContributionAndProof() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapForkInfo(t *testing.T) {
	type args struct {
		from                  *ethpb.Fork
		genesisValidatorsRoot []byte
	}

	tests := []struct {
		name    string
		args    args
		want    *ForkInfo
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				from: &ethpb.Fork{
					PreviousVersion: make([]byte, 4),
					CurrentVersion:  make([]byte, 4),
					Epoch:           0,
				},
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    MockForkInfo(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapForkInfo(tt.args.from, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapForkInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapForkInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapSyncAggregatorSelectionData(t *testing.T) {
	type args struct {
		data *ethpb.SyncAggregatorSelectionData
	}
	tests := []struct {
		name    string
		args    args
		want    *SyncAggregatorSelectionData
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				data: &ethpb.SyncAggregatorSelectionData{
					Slot:              0,
					SubcommitteeIndex: 0,
				},
			},
			want: &SyncAggregatorSelectionData{
				Slot:              "0",
				SubcommitteeIndex: "0",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSyncAggregatorSelectionData(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapSyncAggregatorSelectionData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapSyncAggregatorSelectionData() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapSyncCommitteeMessage(t *testing.T) {
	type args struct {
		message *ethpb.SyncCommitteeMessage
	}
	tests := []struct {
		name    string
		args    args
		want    *SyncCommitteeMessage
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				message: &ethpb.SyncCommitteeMessage{
					Slot:           0,
					BlockRoot:      make([]byte, fieldparams.RootLength),
					ValidatorIndex: 0,
					Signature:      make([]byte, fieldparams.BLSSignatureLength),
				},
			},
			want: &SyncCommitteeMessage{
				Slot:            "0",
				BeaconBlockRoot: hexutil.Encode(make([]byte, fieldparams.RootLength)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSyncCommitteeMessage(tt.args.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapSyncCommitteeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapSyncCommitteeMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
