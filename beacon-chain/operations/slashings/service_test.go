package slashings

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestPool_InsertAttesterSlashing(t *testing.T) {
	type fields struct {
		pending  []*PendingAttesterSlashing
		included map[uint64]bool
	}
	type args struct {
		slashings *ethpb.AttesterSlashing
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*PendingAttesterSlashing
	}{
		{
			name: "Empty list",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1},
						},
					},
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
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 2},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1, 2},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
					},
					validatorToSlash: 1,
				},
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1, 2},
						},
					},
					validatorToSlash: 2,
				},
			},
		},
		{
			name: "Empty list two validators slashed out od three",
			fields: fields{
				pending:  make([]*PendingAttesterSlashing, 0),
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
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
					{
						attesterSlashing: &ethpb.AttesterSlashing{
							Attestation_1: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{1},
							},
							Attestation_2: &ethpb.IndexedAttestation{
								AttestingIndices: []uint64{1},
							},
						},
						validatorToSlash: 1,
					},
				},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{1},
						},
					},
					validatorToSlash: 1,
				},
			},
		},
		{
			name: "Slashing for exited validator ",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{2},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{2},
					},
				},
			},
			want: []*PendingAttesterSlashing{},
		},
		{
			name: "Slashing for futuristic exited validator ",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{4},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{4},
					},
				},
			},
			want: []*PendingAttesterSlashing{
				{
					attesterSlashing: &ethpb.AttesterSlashing{
						Attestation_1: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{4},
						},
						Attestation_2: &ethpb.IndexedAttestation{
							AttestingIndices: []uint64{4},
						},
					},
					validatorToSlash: 4,
				},
			},
		},
		{
			name: "Slashing for slashed validator ",
			fields: fields{
				pending:  []*PendingAttesterSlashing{},
				included: make(map[uint64]bool),
			},
			args: args{
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{5},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{5},
					},
				},
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
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
				},
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
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
				},
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
				slashings: &ethpb.AttesterSlashing{
					Attestation_1: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
					Attestation_2: &ethpb.IndexedAttestation{
						AttestingIndices: []uint64{1},
					},
				},
			},
			want: []*PendingAttesterSlashing{},
		},
		//	{
		//		name: "Maintains sorted order",
		//		fields: fields{
		//			pending: []*ethpb.SignedVoluntaryExit{
		//				{
		//					Exit: &ethpb.VoluntaryExit{
		//						Epoch:          12,
		//						ValidatorIndex: 0,
		//					},
		//				},
		//				{
		//					Exit: &ethpb.VoluntaryExit{
		//						Epoch:          12,
		//						ValidatorIndex: 2,
		//					},
		//				},
		//			},
		//			included: make(map[uint64]bool),
		//		},
		//		args: args{
		//			exit: &ethpb.SignedVoluntaryExit{
		//				Exit: &ethpb.VoluntaryExit{
		//					Epoch:          10,
		//					ValidatorIndex: 1,
		//				},
		//			},
		//		},
		//		want: []*ethpb.SignedVoluntaryExit{
		//			{
		//				Exit: &ethpb.VoluntaryExit{
		//					Epoch:          12,
		//					ValidatorIndex: 0,
		//				},
		//			},
		//			{
		//				Exit: &ethpb.VoluntaryExit{
		//					Epoch:          10,
		//					ValidatorIndex: 1,
		//				},
		//			},
		//			{
		//				Exit: &ethpb.VoluntaryExit{
		//					Epoch:          12,
		//					ValidatorIndex: 2,
		//				},
		//			},
		//		},
		//	},
		//	{
		//		name: "Already included",
		//		fields: fields{
		//			pending: make([]*ethpb.SignedVoluntaryExit, 0),
		//			included: map[uint64]bool{
		//				1: true,
		//			},
		//		},
		//		args: args{
		//			exit: &ethpb.SignedVoluntaryExit{
		//				Exit: &ethpb.VoluntaryExit{
		//					Epoch:          12,
		//					ValidatorIndex: 1,
		//				},
		//			},
		//		},
		//		want: []*ethpb.SignedVoluntaryExit{},
		//	},
	}
	ctx := context.Background()
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				pendingAttesterSlashing: tt.fields.pending,
				included:                tt.fields.included,
			}
			s, err := beaconstate.InitializeFromProtoUnsafe(&p2ppb.BeaconState{Validators: validators})
			if err != nil {
				t.Fatal(err)
			}
			s.SetSlot(16 * params.BeaconConfig().SlotsPerEpoch)
			s.SetSlashings([]uint64{5})
			p.InsertAttesterSlashing(ctx, s, tt.args.slashings)
			if len(p.pendingAttesterSlashing) != len(tt.want) {
				t.Fatalf("Mismatched lengths of pending list. Got %d, wanted %d.", len(p.pendingAttesterSlashing), len(tt.want))
			}
			for i := range p.pendingAttesterSlashing {
				if p.pendingAttesterSlashing[i].validatorToSlash != tt.want[i].validatorToSlash {
					t.Errorf("Pending attester to slash at index %d does not match expected. Got=%v wanted=%v", i, p.pendingAttesterSlashing[i].validatorToSlash, tt.want[i].validatorToSlash)
				}
				if !proto.Equal(p.pendingAttesterSlashing[i].attesterSlashing, tt.want[i].attesterSlashing) {
					t.Errorf("Pending attester slashings at index %d does not match expected. Got=%v wanted=%v", i, p.pendingAttesterSlashing[i], tt.want[i])
				}
			}
		})
	}
}

//func TestPool_MarkIncludedAttesterSlashing(t *testing.T) {
//	type fields struct {
//		pending  []*ethpb.SignedVoluntaryExit
//		included map[uint64]bool
//	}
//	type args struct {
//		exit *ethpb.SignedVoluntaryExit
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   fields
//	}{
//		{
//			name: "Included, does not exist in pending",
//			fields: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 2},
//					},
//				},
//				included: make(map[uint64]bool),
//			},
//			args: args{
//				exit: &ethpb.SignedVoluntaryExit{
//					Exit: &ethpb.VoluntaryExit{ValidatorIndex: 3},
//				},
//			},
//			want: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 2},
//					},
//				},
//				included: map[uint64]bool{
//					3: true,
//				},
//			},
//		},
//		{
//			name: "Removes from pending list",
//			fields: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 1},
//					},
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 2},
//					},
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 3},
//					},
//				},
//				included: map[uint64]bool{
//					0: true,
//				},
//			},
//			args: args{
//				exit: &ethpb.SignedVoluntaryExit{
//					Exit: &ethpb.VoluntaryExit{ValidatorIndex: 2},
//				},
//			},
//			want: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 1},
//					},
//					{
//						Exit: &ethpb.VoluntaryExit{ValidatorIndex: 3},
//					},
//				},
//				included: map[uint64]bool{
//					0: true,
//					2: true,
//				},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Pool{
//				pending:  tt.fields.pending,
//				included: tt.fields.included,
//			}
//			p.MarkIncluded(tt.args.exit)
//			if len(p.pending) != len(tt.want.pending) {
//				t.Fatalf("Mismatched lengths of pending list. Got %d, wanted %d.", len(p.pending), len(tt.want.pending))
//			}
//			for i := range p.pending {
//				if !proto.Equal(p.pending[i], tt.want.pending[i]) {
//					t.Errorf("Pending exit at index %d does not match expected. Got=%v wanted=%v", i, p.pending[i], tt.want.pending[i])
//				}
//			}
//			if !reflect.DeepEqual(p.included, tt.want.included) {
//				t.Errorf("Included map is not as expected. Got=%v wanted=%v", p.included, tt.want.included)
//			}
//		})
//	}
//}
//
//func TestPool_PendingAttesterSlashings(t *testing.T) {
//	type fields struct {
//		pending []*ethpb.SignedVoluntaryExit
//	}
//	type args struct {
//		slot uint64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   []*ethpb.SignedVoluntaryExit
//	}{
//		{
//			name: "Empty list",
//			fields: fields{
//				pending: []*ethpb.SignedVoluntaryExit{},
//			},
//			args: args{
//				slot: 100000,
//			},
//			want: []*ethpb.SignedVoluntaryExit{},
//		},
//		{
//			name: "All eligible",
//			fields: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{Exit: &ethpb.VoluntaryExit{Epoch: 0}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 1}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 2}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 3}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 4}},
//				},
//			},
//			args: args{
//				slot: 1000000,
//			},
//			want: []*ethpb.SignedVoluntaryExit{
//				{Exit: &ethpb.VoluntaryExit{Epoch: 0}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 1}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 2}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 3}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 4}},
//			},
//		},
//		{
//			name: "All eligible, more than max",
//			fields: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{Exit: &ethpb.VoluntaryExit{Epoch: 0}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 1}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 2}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 3}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 4}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 5}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 6}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 7}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 8}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 9}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 10}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 11}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 12}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 13}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 14}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 15}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 16}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 17}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 18}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 19}},
//				},
//			},
//			args: args{
//				slot: 1000000,
//			},
//			want: []*ethpb.SignedVoluntaryExit{
//				{Exit: &ethpb.VoluntaryExit{Epoch: 0}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 1}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 2}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 3}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 4}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 5}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 6}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 7}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 8}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 9}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 10}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 11}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 12}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 13}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 14}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 15}},
//			},
//		},
//		{
//			name: "Some eligible",
//			fields: fields{
//				pending: []*ethpb.SignedVoluntaryExit{
//					{Exit: &ethpb.VoluntaryExit{Epoch: 0}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 3}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 4}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 2}},
//					{Exit: &ethpb.VoluntaryExit{Epoch: 1}},
//				},
//			},
//			args: args{
//				slot: 2 * params.BeaconConfig().SlotsPerEpoch,
//			},
//			want: []*ethpb.SignedVoluntaryExit{
//				{Exit: &ethpb.VoluntaryExit{Epoch: 0}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 2}},
//				{Exit: &ethpb.VoluntaryExit{Epoch: 1}},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Pool{
//				pending: tt.fields.pending,
//			}
//			if got := p.PendingExits(tt.args.slot); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("PendingExits() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
