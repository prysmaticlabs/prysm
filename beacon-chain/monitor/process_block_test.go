package monitor

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProcessSlashings(t *testing.T) {
	tests := []struct {
		name      string
		block     *ethpb.BeaconBlock
		wantedErr string
	}{
		{
			name: "Proposer slashing a tracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 2,
									Slot:          params.BeaconConfig().SlotsPerEpoch + 1,
								},
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 2,
									Slot:          0,
								},
							},
						},
					},
				},
			},
			wantedErr: "\"Proposer slashing was included\" ProposerIndex=2",
		},
		{
			name: "Proposer slashing an untracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					ProposerSlashings: []*ethpb.ProposerSlashing{
						{
							Header_1: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 3,
									Slot:          params.BeaconConfig().SlotsPerEpoch + 4,
								},
							},
							Header_2: &ethpb.SignedBeaconBlockHeader{
								Header: &ethpb.BeaconBlockHeader{
									ProposerIndex: 3,
									Slot:          0,
								},
							},
						},
					},
				},
			},
			wantedErr: "",
		},
		{
			name: "Attester slashing a tracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					AttesterSlashings: []*ethpb.AttesterSlashing{
						{
							Attestation_1: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Source: &ethpb.Checkpoint{Epoch: 1},
								},
								AttestingIndices: []uint64{1, 3, 4},
							}),
							Attestation_2: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								AttestingIndices: []uint64{1, 5, 6},
							}),
						},
					},
				},
			},
			wantedErr: "\"Attester slashing was included\" AttesterIndex=1",
		},
		{
			name: "Attester slashing untracked index",
			block: &ethpb.BeaconBlock{
				Body: &ethpb.BeaconBlockBody{
					AttesterSlashings: []*ethpb.AttesterSlashing{
						{
							Attestation_1: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Source: &ethpb.Checkpoint{Epoch: 1},
								},
								AttestingIndices: []uint64{1, 3, 4},
							}),
							Attestation_2: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
								AttestingIndices: []uint64{3, 5, 6},
							}),
						},
					},
				},
			},
			wantedErr: "",
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			s := &Service{
				config: &ValidatorMonitorConfig{
					TrackedValidators: map[types.ValidatorIndex]interface{}{
						1: nil,
						2: nil,
					},
				},
			}
			s.processSlashings(wrapper.WrappedPhase0BeaconBlock(tt.block))
			if tt.wantedErr != "" {
				require.LogsContain(t, hook, tt.wantedErr)
			} else {
				require.LogsDoNotContain(t, hook, "slashing")
			}
		})
	}
}
