package slasher

import (
	"reflect"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_groupByValidatorChunkIndex(t *testing.T) {
	tests := []struct {
		name   string
		params *Parameters
		atts   []*slashertypes.IndexedAttestationWrapper
		want   map[uint64][]*slashertypes.IndexedAttestationWrapper
	}{
		{
			name:   "No attestations returns empty map",
			params: DefaultParams(),
			atts:   make([]*slashertypes.IndexedAttestationWrapper, 0),
			want:   make(map[uint64][]*slashertypes.IndexedAttestationWrapper),
		},
		{
			name: "Groups multiple attestations belonging to single validator chunk",
			params: &Parameters{
				validatorChunkSize: 2,
			},
			atts: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				createAttestationWrapper(0, 0, []uint64{0, 1} /* indices */, nil /* signingRoot */),
			},
			want: map[uint64][]*slashertypes.IndexedAttestationWrapper{
				0: {
					createAttestationWrapper(0, 0, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 0, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
			},
		},
		{
			name: "Groups single attestation belonging to multiple validator chunk",
			params: &Parameters{
				validatorChunkSize: 2,
			},
			atts: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, []uint64{0, 2, 4} /* indices */, nil /* signingRoot */),
			},
			want: map[uint64][]*slashertypes.IndexedAttestationWrapper{
				0: {
					createAttestationWrapper(0, 0, []uint64{0, 2, 4} /* indices */, nil /* signingRoot */),
				},
				1: {
					createAttestationWrapper(0, 0, []uint64{0, 2, 4} /* indices */, nil /* signingRoot */),
				},
				2: {
					createAttestationWrapper(0, 0, []uint64{0, 2, 4} /* indices */, nil /* signingRoot */),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				params: tt.params,
			}
			if got := s.groupByValidatorChunkIndex(tt.atts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByValidatorChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_groupByChunkIndex(t *testing.T) {
	tests := []struct {
		name   string
		params *Parameters
		atts   []*slashertypes.IndexedAttestationWrapper
		want   map[uint64][]*slashertypes.IndexedAttestationWrapper
	}{
		{
			name:   "No attestations returns empty map",
			params: DefaultParams(),
			atts:   make([]*slashertypes.IndexedAttestationWrapper, 0),
			want:   make(map[uint64][]*slashertypes.IndexedAttestationWrapper),
		},
		{
			name: "Groups multiple attestations belonging to single chunk",
			params: &Parameters{
				chunkSize:     2,
				historyLength: 3,
			},
			atts: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */),
				createAttestationWrapper(1, 0, nil /* indices */, nil /* signingRoot */),
			},
			want: map[uint64][]*slashertypes.IndexedAttestationWrapper{
				0: {
					createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */),
					createAttestationWrapper(1, 0, nil /* indices */, nil /* signingRoot */),
				},
			},
		},
		{
			name: "Groups multiple attestations belonging to multiple chunks",
			params: &Parameters{
				chunkSize:     2,
				historyLength: 3,
			},
			atts: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */),
				createAttestationWrapper(1, 0, nil /* indices */, nil /* signingRoot */),
				createAttestationWrapper(2, 0, nil /* indices */, nil /* signingRoot */),
			},
			want: map[uint64][]*slashertypes.IndexedAttestationWrapper{
				0: {
					createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */),
					createAttestationWrapper(1, 0, nil /* indices */, nil /* signingRoot */),
				},
				1: {
					createAttestationWrapper(2, 0, nil /* indices */, nil /* signingRoot */),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				params: tt.params,
			}
			if got := s.groupByChunkIndex(tt.atts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_validateAttestationIntegrity(t *testing.T) {
	tests := []struct {
		name           string
		input          []*slashertypes.IndexedAttestationWrapper
		inputEpoch     types.Epoch
		wantedValid    []*slashertypes.IndexedAttestationWrapper
		wantedDeferred []*slashertypes.IndexedAttestationWrapper
		wantedDropped  int
	}{
		{
			name:          "Nil attestation input gets dropped",
			input:         make([]*slashertypes.IndexedAttestationWrapper, 1),
			inputEpoch:    0,
			wantedDropped: 1,
		},
		{
			name: "Nil attestation data gets dropped",
			input: []*slashertypes.IndexedAttestationWrapper{
				{
					IndexedAttestation: &ethpb.IndexedAttestation{},
				},
			},
			inputEpoch:    0,
			wantedDropped: 1,
		},
		{
			name: "Nil attestation source and target gets dropped",
			input: []*slashertypes.IndexedAttestationWrapper{
				{
					IndexedAttestation: &ethpb.IndexedAttestation{
						Data: &ethpb.AttestationData{},
					},
				},
			},
			inputEpoch:    0,
			wantedDropped: 1,
		},
		{
			name: "Nil attestation source and good target gets dropped",
			input: []*slashertypes.IndexedAttestationWrapper{
				{
					IndexedAttestation: &ethpb.IndexedAttestation{
						Data: &ethpb.AttestationData{
							Target: &ethpb.Checkpoint{},
						},
					},
				},
			},
			inputEpoch:    0,
			wantedDropped: 1,
		},
		{
			name: "Nil attestation target and good source gets dropped",
			input: []*slashertypes.IndexedAttestationWrapper{
				{
					IndexedAttestation: &ethpb.IndexedAttestation{
						Data: &ethpb.AttestationData{
							Source: &ethpb.Checkpoint{},
						},
					},
				},
			},
			inputEpoch:    0,
			wantedDropped: 1,
		},
		{
			name: "Source > target gets dropped",
			input: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(1, 0, []uint64{1}, make([]byte, 32)),
			},
			inputEpoch:    0,
			wantedDropped: 1,
		},
		{
			name: "Source < target is valid",
			input: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, make([]byte, 32)),
			},
			inputEpoch: 1,
			wantedValid: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 1, []uint64{1}, make([]byte, 32)),
			},
			wantedDropped: 0,
		},
		{
			name: "Source == target is valid",
			input: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, []uint64{1}, make([]byte, 32)),
			},
			inputEpoch: 1,
			wantedValid: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 0, []uint64{1}, make([]byte, 32)),
			},
			wantedDropped: 0,
		},
		{
			name: "Attestation from the future is deferred",
			input: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 2, []uint64{1}, make([]byte, 32)),
			},
			inputEpoch: 1,
			wantedDeferred: []*slashertypes.IndexedAttestationWrapper{
				createAttestationWrapper(0, 2, []uint64{1}, make([]byte, 32)),
			},
			wantedDropped: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Service{
				params: DefaultParams(),
			}
			valid, deferred, numDropped := srv.validateAttestationIntegrity(tt.input, tt.inputEpoch)
			if len(tt.wantedValid) > 0 {
				require.DeepEqual(t, tt.wantedValid, valid)
			}
			if len(tt.wantedDeferred) > 0 {
				require.DeepEqual(t, tt.wantedDeferred, deferred)
			}
			require.DeepEqual(t, tt.wantedDropped, numDropped)
		})
	}
}

func Test_logSlashingEvent(t *testing.T) {
	tests := []struct {
		name     string
		slashing *slashertypes.Slashing
		want     string
		noLog    bool
	}{
		{
			name: "Surrounding vote",
			slashing: &slashertypes.Slashing{
				Kind:            slashertypes.SurroundingVote,
				PrevAttestation: createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
				Attestation:     createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
			},
			want: "Attester surrounding vote",
		},
		{
			name: "Surrounded vote",
			slashing: &slashertypes.Slashing{
				Kind:            slashertypes.SurroundedVote,
				PrevAttestation: createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
				Attestation:     createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
			},
			want: "Attester surrounded vote",
		},
		{
			name: "Double vote",
			slashing: &slashertypes.Slashing{
				Kind:            slashertypes.DoubleVote,
				PrevAttestation: createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
				Attestation:     createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
			},
			want: "Attester double vote",
		},
		{
			name: "Not slashable",
			slashing: &slashertypes.Slashing{
				Kind:            slashertypes.NotSlashable,
				PrevAttestation: createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
				Attestation:     createAttestationWrapper(0, 0, nil /* indices */, nil /* signingRoot */).IndexedAttestation,
			},
			noLog: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			logSlashingEvent(tt.slashing)
			if tt.noLog {
				require.LogsDoNotContain(t, hook, "slashing")
			} else {
				require.LogsContain(t, hook, tt.want)
			}
		})
	}
}

func Test_isDoubleProposal(t *testing.T) {
	type args struct {
		incomingSigningRoot [32]byte
		existingSigningRoot [32]byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Existing signing root empty returns false",
			args: args{
				incomingSigningRoot: [32]byte{1},
				existingSigningRoot: params.BeaconConfig().ZeroHash,
			},
			want: false,
		},
		{
			name: "Existing signing root non-empty and equal to incoming returns false",
			args: args{
				incomingSigningRoot: [32]byte{1},
				existingSigningRoot: [32]byte{1},
			},
			want: false,
		},
		{
			name: "Existing signing root non-empty and not-equal to incoming returns true",
			args: args{
				incomingSigningRoot: [32]byte{1},
				existingSigningRoot: [32]byte{2},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDoubleProposal(tt.args.incomingSigningRoot, tt.args.existingSigningRoot); got != tt.want {
				t.Errorf("isDoubleProposal() = %v, want %v", got, tt.want)
			}
		})
	}
}
