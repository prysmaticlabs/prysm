package copyutil

import (
	"reflect"
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
)

func TestCopyAttestation(t *testing.T) {
	att := genAtt()
	copiedAtt := CopyAttestation(att)

	v := reflect.ValueOf(copiedAtt).Elem()
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() == reflect.Slice {
			// Check length
		}
	}
}

func TestCopyAttestationData(t *testing.T) {
	type args struct {
		attData *ethpb.AttestationData
	}
	tests := []struct {
		name string
		args args
		want *ethpb.AttestationData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyAttestationData(tt.args.attData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyAttestationData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyAttestations(t *testing.T) {
	type args struct {
		attestations []*ethpb.Attestation
	}
	tests := []struct {
		name string
		args args
		want []*ethpb.Attestation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyAttestations(tt.args.attestations); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyAttestations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyAttesterSlashings(t *testing.T) {
	type args struct {
		slashings []*ethpb.AttesterSlashing
	}
	tests := []struct {
		name string
		args args
		want []*ethpb.AttesterSlashing
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyAttesterSlashings(tt.args.slashings); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyAttesterSlashings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyBeaconBlock(t *testing.T) {
	type args struct {
		block *ethpb.BeaconBlock
	}
	tests := []struct {
		name string
		args args
		want *ethpb.BeaconBlock
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyBeaconBlock(tt.args.block); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyBeaconBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyBeaconBlockBody(t *testing.T) {
	type args struct {
		body *ethpb.BeaconBlockBody
	}
	tests := []struct {
		name string
		args args
		want *ethpb.BeaconBlockBody
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyBeaconBlockBody(tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyBeaconBlockBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyBeaconBlockHeader(t *testing.T) {
	type args struct {
		header *ethpb.BeaconBlockHeader
	}
	tests := []struct {
		name string
		args args
		want *ethpb.BeaconBlockHeader
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyBeaconBlockHeader(tt.args.header); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyBeaconBlockHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyCheckpoint(t *testing.T) {
	type args struct {
		cp *ethpb.Checkpoint
	}
	tests := []struct {
		name string
		args args
		want *ethpb.Checkpoint
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyCheckpoint(tt.args.cp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyCheckpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyDeposit(t *testing.T) {
	type args struct {
		deposit *ethpb.Deposit
	}
	tests := []struct {
		name string
		args args
		want *ethpb.Deposit
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyDeposit(tt.args.deposit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyDeposit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyDepositData(t *testing.T) {
	type args struct {
		depData *ethpb.Deposit_Data
	}
	tests := []struct {
		name string
		args args
		want *ethpb.Deposit_Data
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyDepositData(tt.args.depData); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyDepositData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyDeposits(t *testing.T) {
	type args struct {
		deposits []*ethpb.Deposit
	}
	tests := []struct {
		name string
		args args
		want []*ethpb.Deposit
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyDeposits(tt.args.deposits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyDeposits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyETH1Data(t *testing.T) {
	type args struct {
		data *ethpb.Eth1Data
	}
	tests := []struct {
		name string
		args args
		want *ethpb.Eth1Data
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyETH1Data(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyETH1Data() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyIndexedAttestation(t *testing.T) {
	type args struct {
		indexedAtt *ethpb.IndexedAttestation
	}
	tests := []struct {
		name string
		args args
		want *ethpb.IndexedAttestation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyIndexedAttestation(tt.args.indexedAtt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyIndexedAttestation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyPendingAttestation(t *testing.T) {
	type args struct {
		att *pbp2p.PendingAttestation
	}
	tests := []struct {
		name string
		args args
		want *pbp2p.PendingAttestation
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyPendingAttestation(tt.args.att); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyPendingAttestation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyProposerSlashing(t *testing.T) {
	type args struct {
		slashing *ethpb.ProposerSlashing
	}
	tests := []struct {
		name string
		args args
		want *ethpb.ProposerSlashing
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyProposerSlashing(tt.args.slashing); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyProposerSlashing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyProposerSlashings(t *testing.T) {
	type args struct {
		slashings []*ethpb.ProposerSlashing
	}
	tests := []struct {
		name string
		args args
		want []*ethpb.ProposerSlashing
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyProposerSlashings(tt.args.slashings); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyProposerSlashings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopySignedBeaconBlock(t *testing.T) {
	type args struct {
		sigBlock *ethpb.SignedBeaconBlock
	}
	tests := []struct {
		name string
		args args
		want *ethpb.SignedBeaconBlock
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopySignedBeaconBlock(tt.args.sigBlock); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopySignedBeaconBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopySignedBeaconBlockHeader(t *testing.T) {
	type args struct {
		header *ethpb.SignedBeaconBlockHeader
	}
	tests := []struct {
		name string
		args args
		want *ethpb.SignedBeaconBlockHeader
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopySignedBeaconBlockHeader(tt.args.header); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopySignedBeaconBlockHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopySignedVoluntaryExit(t *testing.T) {
	type args struct {
		exit *ethpb.SignedVoluntaryExit
	}
	tests := []struct {
		name string
		args args
		want *ethpb.SignedVoluntaryExit
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopySignedVoluntaryExit(tt.args.exit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopySignedVoluntaryExit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopySignedVoluntaryExits(t *testing.T) {
	type args struct {
		exits []*ethpb.SignedVoluntaryExit
	}
	tests := []struct {
		name string
		args args
		want []*ethpb.SignedVoluntaryExit
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopySignedVoluntaryExits(tt.args.exits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopySignedVoluntaryExits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopySyncCommitteeContribution(t *testing.T) {
	type args struct {
		c *prysmv2.SyncCommitteeContribution
	}
	tests := []struct {
		name string
		args args
		want *prysmv2.SyncCommitteeContribution
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopySyncCommitteeContribution(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopySyncCommitteeContribution() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyValidator(t *testing.T) {
	type args struct {
		val *ethpb.Validator
	}
	tests := []struct {
		name string
		args args
		want *ethpb.Validator
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyValidator(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

var bytes = []byte{'a'}

func genAtt() *ethpb.Attestation {
	return &ethpb.Attestation{
		AggregationBits: bytes,
		Data:            genAttData(),
		Signature:       bytes,
	}
}

func genAttData() *ethpb.AttestationData {
	return &ethpb.AttestationData{
		Slot:            1,
		CommitteeIndex:  2,
		BeaconBlockRoot: bytes,
		Source:          genCheckpoint(),
		Target:          genCheckpoint(),
	}
}

func genCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: 1,
		Root:  bytes,
	}
}
