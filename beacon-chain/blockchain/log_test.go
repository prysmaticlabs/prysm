package blockchain

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_logStateTransitionData(t *testing.T) {
	tests := []struct {
		name string
		b    *ethpb.BeaconBlock
		want string
	}{
		{name: "empty block body",
			b:    &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}},
			want: "\"Finished applying state transition\" prefix=blockchain slot=0",
		},
		{name: "has attestation",
			b:    &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{Attestations: []*ethpb.Attestation{{}}}},
			want: "\"Finished applying state transition\" attestations=1 prefix=blockchain slot=0",
		},
		{name: "has deposit",
			b: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{
				Attestations: []*ethpb.Attestation{{}},
				Deposits:     []*ethpb.Deposit{{}}}},
			want: "\"Finished applying state transition\" attestations=1 deposits=1 prefix=blockchain slot=0",
		},
		{name: "has attester slashing",
			b: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{
				AttesterSlashings: []*ethpb.AttesterSlashing{{}}}},
			want: "\"Finished applying state transition\" attesterSlashings=1 prefix=blockchain slot=0",
		},
		{name: "has proposer slashing",
			b: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{
				ProposerSlashings: []*ethpb.ProposerSlashing{{}}}},
			want: "\"Finished applying state transition\" prefix=blockchain proposerSlashings=1 slot=0",
		},
		{name: "has exit",
			b: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{
				VoluntaryExits: []*ethpb.SignedVoluntaryExit{{}}}},
			want: "\"Finished applying state transition\" prefix=blockchain slot=0 voluntaryExits=1",
		},
		{name: "has everything",
			b: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{
				Attestations:      []*ethpb.Attestation{{}},
				Deposits:          []*ethpb.Deposit{{}},
				AttesterSlashings: []*ethpb.AttesterSlashing{{}},
				ProposerSlashings: []*ethpb.ProposerSlashing{{}},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{{}}}},
			want: "\"Finished applying state transition\" attestations=1 attesterSlashings=1 deposits=1 prefix=blockchain proposerSlashings=1 slot=0 voluntaryExits=1",
		},
	}
	for _, tt := range tests {
		hook := logTest.NewGlobal()
		t.Run(tt.name, func(t *testing.T) {
			logStateTransitionData(tt.b)
			require.LogsContain(t, hook, tt.want)
		})
	}
}
