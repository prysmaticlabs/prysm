package v1

import (
	"reflect"
	"testing"

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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapAggregateAndProof(tt.args.from)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAggregateAndProof() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapAggregateAndProof() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapAttesterSlashing(tt.args.slashing)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAttesterSlashing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapAttesterSlashing() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapBeaconBlockAltair(tt.args.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapBeaconBlockAltair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapBeaconBlockAltair() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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

func TestMapBeaconBlockBodyAltair(t *testing.T) {
	type args struct {
		body *ethpb.BeaconBlockBodyAltair
	}
	tests := []struct {
		name    string
		args    args
		want    *BeaconBlockBodyAltair
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapBeaconBlockBodyAltair(tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapBeaconBlockBodyAltair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapBeaconBlockBodyAltair() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapCheckPoint(t *testing.T) {
	type args struct {
		checkpoint *ethpb.Checkpoint
	}
	tests := []struct {
		name    string
		args    args
		want    *Checkpoint
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapCheckPoint(tt.args.checkpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapCheckPoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapCheckPoint() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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

func TestMapDeposit(t *testing.T) {
	type args struct {
		deposit *ethpb.Deposit
	}
	tests := []struct {
		name    string
		args    args
		want    *Deposit
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapDeposit(tt.args.deposit)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapDeposit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapDeposit() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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

func TestMapIndexedAttestation(t *testing.T) {
	type args struct {
		attestation *ethpb.IndexedAttestation
	}
	tests := []struct {
		name    string
		args    args
		want    *IndexedAttestation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapIndexedAttestation(tt.args.attestation)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapIndexedAttestation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapIndexedAttestation() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapProposerSlashing(t *testing.T) {
	type args struct {
		slashing *ethpb.ProposerSlashing
	}
	tests := []struct {
		name    string
		args    args
		want    *ProposerSlashing
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapProposerSlashing(tt.args.slashing)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapProposerSlashing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapProposerSlashing() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapSignedBeaconBlockHeader(t *testing.T) {
	type args struct {
		signedHeader *ethpb.SignedBeaconBlockHeader
	}
	tests := []struct {
		name    string
		args    args
		want    *SignedBeaconBlockHeader
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSignedBeaconBlockHeader(tt.args.signedHeader)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapSignedBeaconBlockHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapSignedBeaconBlockHeader() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapSignedVoluntaryExit(t *testing.T) {
	type args struct {
		signedVoluntaryExit *ethpb.SignedVoluntaryExit
	}
	tests := []struct {
		name    string
		args    args
		want    *SignedVoluntaryExit
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSignedVoluntaryExit(tt.args.signedVoluntaryExit)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapSignedVoluntaryExit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapSignedVoluntaryExit() got = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
