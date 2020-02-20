package slashings

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func attesterSlashingForValIdx(valIdx ...uint64) *ethpb.AttesterSlashing {
	return &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			AttestingIndices: valIdx,
		},
		Attestation_2: &ethpb.IndexedAttestation{
			AttestingIndices: valIdx,
		},
	}
}

func pendingSlashingForValIdx(valIdx ...uint64) *PendingAttesterSlashing {
	return &PendingAttesterSlashing{
		attesterSlashing: attesterSlashingForValIdx(valIdx...),
		validatorToSlash: valIdx[0],
	}
}

func generateNPendingSlashings(n uint64) []*PendingAttesterSlashing {
	pendingAttSlashings := make([]*PendingAttesterSlashing, n)
	for i := uint64(0); i < n; i++ {
		pendingAttSlashings[i] = &PendingAttesterSlashing{
			attesterSlashing: attesterSlashingForValIdx(i),
			validatorToSlash: i,
		}
	}
	return pendingAttSlashings
}

func generateNAttSlashings(n uint64) []*ethpb.AttesterSlashing {
	attSlashings := make([]*ethpb.AttesterSlashing, n)
	for i := uint64(0); i < n; i++ {
		attSlashings[i] = attesterSlashingForValIdx(i)
	}
	return attSlashings
}

func TestPool_InsertAttesterSlashing(t *testing.T) {
	type fields struct {
		pending  []*PendingAttesterSlashing
		included map[uint64]bool
		wantErr  bool
		err      string
	}
	type args struct {
		slashing *ethpb.AttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*PendingAttesterSlashing
		err    string
	}{
		{
			name: "Empty list",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: attesterSlashingForValIdx(1),
					validatorToSlash: 1,
				},
			},
		},
		{
			name: "Empty list two validators slashed",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{0, 1},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{0, 1},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{0, 1},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{0, 1},
						},
					},
					validatorToSlash: 0,
				},
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{0, 1},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{0, 1},
						},
					},
					validatorToSlash: 1,
				},
			},
		},
		{
			name: "Empty list two validators slashed out of three",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 2, 3},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 3},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2, 3},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 3},
						},
					},
					validatorToSlash: 1,
				},
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2, 3},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 3},
						},
					},
					validatorToSlash: 3,
				},
			},
		},
		{
			name: "Duplicate identical slashing",
			fields: fields{
				pending: []*PendingAttesterSlashing{

					pendingSlashingForValIdx(1),
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{

				pendingSlashingForValIdx(1),
			},
		},
		{
			name: "Slashing for exited validator",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(2),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Slashing for futuristic exited validator",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(4),
			},
			want: []*PendingAttesterSlashing{
				pendingSlashingForValIdx(4),
			},
		},
		{
			name: "Slashing for slashed validator",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
				wantErr:  true,
				err:      "cannot be slashed",
			},
			args: args{
				slashing: attesterSlashingForValIdx(5),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Already included",
			fields: fields{
				pending: []*PendingAttesterSlashing{},
				included: map[uint64]bool{
					1: true,
				},
			},
			args: args{
				slashing: attesterSlashingForValIdx(1),
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Maintains sorted order",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					pendingSlashingForValIdx(0),
					pendingSlashingForValIdx(2),
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(1),
			},
			want: generateNPendingSlashings(3),
		},
	}
	validators := []*ethpb.Validator{
		{ // 0
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{ // 1
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{ // 2 - Already exited.
			ExitEpoch: 15,
		},
		{ // 3
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{ // 4 - Will be exited.
			ExitEpoch: 17,
		},
		{ // 5 - Slashed.
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
				included:                tt.fields.included,
			}
			s, err := beaconstate.InitializeFromProtoUnsafe(&p2ppb.BeaconState{
				Slot:       16 * params.BeaconConfig().SlotsPerEpoch,
				Validators: validators,
			})
			if err != nil {
				t.Fatal(err)
			}
			err = p.InsertAttesterSlashing(s, tt.args.slashing)
			if err != nil && tt.fields.wantErr && !strings.Contains(err.Error(), tt.fields.err) {
				t.Fatalf("Wanted err: %v, received %v", tt.fields.err, err)
			}
			if len(p.pendingAttesterSlashing) != len(tt.want) {
				t.Fatalf("Mismatched lengths of pending list. Got %d, wanted %d.", len(p.pendingAttesterSlashing), len(tt.want))
			}
			for i := range p.pendingAttesterSlashing {
				if p.pendingAttesterSlashing[i].validatorToSlash != tt.want[i].validatorToSlash {
					t.Errorf(
						"Pending attester to slash at index %d does not match expected. Got=%v wanted=%v",
						i,
						p.pendingAttesterSlashing[i].validatorToSlash,
						tt.want[i].validatorToSlash,
					)
				}
				if !proto.Equal(p.pendingAttesterSlashing[i].attesterSlashing, tt.want[i].attesterSlashing) {
					t.Errorf(
						"Pending attester slashing at index %d does not match expected. Got=%v wanted=%v",
						i,
						p.pendingAttesterSlashing[i],
						tt.want[i],
					)
				}
			}
		})
	}
}

func TestPool_MarkIncludedAttesterSlashing(t *testing.T) {
	type fields struct {
		pending  []*PendingAttesterSlashing
		included map[uint64]bool
	}
	type args struct {
		slashing *ethpb.AttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   fields
	}{
		{
			name: "Included, does not exist in pending",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(1),
						validatorToSlash: 1,
					},
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashing: attesterSlashingForValIdx(3),
			},
			want: fields{
				pending: []*PendingAttesterSlashing{
					pendingSlashingForValIdx(1),
				},
				included: map[uint64]bool{
					3: true,
				},
			},
		},
		{
			name: "Removes from pending list",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					pendingSlashingForValIdx(1),
					pendingSlashingForValIdx(2),
					pendingSlashingForValIdx(3),
				},
				included: map[uint64]bool{
					0: true,
				},
			},
			args: args{
				slashing: attesterSlashingForValIdx(2),
			},
			want: fields{
				pending: []*PendingAttesterSlashing{
					pendingSlashingForValIdx(1),
					pendingSlashingForValIdx(3),
				},
				included: map[uint64]bool{
					0: true,
					2: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
				included:                tt.fields.included,
			}
			p.MarkIncludedAttesterSlashing(tt.args.slashing)
			if len(p.pendingAttesterSlashing) != len(tt.want.pending) {
				t.Fatalf(
					"Mismatched lengths of pending list. Got %d, wanted %d.",
					len(p.pendingAttesterSlashing),
					len(tt.want.pending),
				)
			}
			for i := range p.pendingAttesterSlashing {
				if !reflect.DeepEqual(p.pendingAttesterSlashing[i], tt.want.pending[i]) {
					t.Errorf(
						"Pending attester slashing at index %d does not match expected. Got=%v wanted=%v",
						i,
						p.pendingAttesterSlashing[i],
						tt.want.pending[i],
					)
				}
			}
			if !reflect.DeepEqual(p.included, tt.want.included) {
				t.Errorf("Included map is not as expected. Got=%v wanted=%v", p.included, tt.want.included)
			}
		})
	}
}

func TestPool_PendingAttesterSlashings(t *testing.T) {
	type fields struct {
		pending []*PendingAttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		want   []*ethpb.AttesterSlashing
	}{
		{
			name: "Empty list",
			fields: fields{
				pending: []*PendingAttesterSlashing{},
			},
			want: []*ethpb.AttesterSlashing{},
		},
		{
			name: "All eligible",
			fields: fields{
				pending: generateNPendingSlashings(1),
			},
			want: generateNAttSlashings(1),
		},
		{
			name: "Multiple indices",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					pendingSlashingForValIdx(1, 5, 8),
				},
			},
			want: []*ethpb.AttesterSlashing{
				attesterSlashingForValIdx(1, 5, 8),
			},
		},
		{
			name: "All eligible, over max",
			fields: fields{
				pending: generateNPendingSlashings(6),
			},
			want: generateNAttSlashings(1),
		},
		{
			name: "No duplicate slashings for grouped",
			fields: fields{
				pending: generateNPendingSlashings(16),
			},
			want: generateNAttSlashings(1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
			}
			if got := p.PendingAttesterSlashings(); !reflect.DeepEqual(tt.want, got) {
				t.Errorf("Unexpected return from PendingAttesterSlashings, wanted %v, received %v", tt.want, got)
			}
		})
	}
}

func TestPool_PendingAttesterSlashings_2Max(t *testing.T) {
	conf := params.BeaconConfig()
	conf.MaxAttesterSlashings = 2
	params.OverrideBeaconConfig(conf)

	type fields struct {
		pending []*PendingAttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		want   []*ethpb.AttesterSlashing
	}{
		{
			name: "No duplicates with grouped att slashings",
			fields: fields{
				pending: []*PendingAttesterSlashing{
					{
						attesterSlashing: attesterSlashingForValIdx(4, 12, 40),
						validatorToSlash: 4,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(6, 8, 24),
						validatorToSlash: 6,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(6, 8, 24),
						validatorToSlash: 8,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(4, 12, 40),
						validatorToSlash: 12,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(6, 8, 24),
						validatorToSlash: 24,
					},
					{
						attesterSlashing: attesterSlashingForValIdx(4, 12, 40),
						validatorToSlash: 40,
					},
				},
			},
			want: []*ethpb.AttesterSlashing{
				attesterSlashingForValIdx(4, 12, 40),
				attesterSlashingForValIdx(6, 8, 24),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
			}
			if got := p.PendingAttesterSlashings(); !reflect.DeepEqual(tt.want, got) {
				t.Errorf("Unexpected return from PendingAttesterSlashings, wanted %v, received %v", tt.want, got)
			}
		})
	}
}
