package attaggregation

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
)

func TestMaxCoverAttestationAggregation_newMaxCoverProblem(t *testing.T) {
	type args struct {
		atts []*ethpb.Attestation
		k    int
	}
	tests := []struct {
		name        string
		args        args
		want        *maxCoverProblem
		wantErr     bool
		expectedErr error
	}{
		{
			name: "nil attestations",
			args: args{
				atts: nil,
			},
			want:        nil,
			wantErr:     true,
			expectedErr: ErrInvalidAttestationCount,
		},
		{
			name: "no attestations",
			args: args{
				atts: []*ethpb.Attestation{},
			},
			want:        nil,
			wantErr:     true,
			expectedErr: ErrInvalidAttestationCount,
		},
		{
			name: "attestations of different bitlist length",
			args: args{
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.NewBitlist(64)},
					{AggregationBits: bitfield.NewBitlist(128)},
				},
			},
			want:        nil,
			wantErr:     true,
			expectedErr: ErrBitsDifferentLen,
		},
		{
			name: "single attestation",
			args: args{
				k: 1,
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.Bitlist{0b00001010, 0b1}},
				},
			},
			want: &maxCoverProblem{
				k: 1,
				candidates: maxCoverCandidateList{
					&maxCoverCandidate{
						key:       0,
						bits:      &bitfield.Bitlist{0b00001010, 0b1},
						score:     2,
						processed: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple attestations",
			args: args{
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.Bitlist{0b00001010, 0b1}},
					{AggregationBits: bitfield.Bitlist{0b00101010, 0b1}},
					{AggregationBits: bitfield.Bitlist{0b11111010, 0b1}},
					{AggregationBits: bitfield.Bitlist{0b00000010, 0b1}},
					{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				},
				k: 50,
			},
			want: &maxCoverProblem{
				k: 5,
				candidates: maxCoverCandidateList{
					{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
					{1, &bitfield.Bitlist{0b00101010, 0b1}, 3, false},
					{2, &bitfield.Bitlist{0b11111010, 0b1}, 6, false},
					{3, &bitfield.Bitlist{0b00000010, 0b1}, 1, false},
					{4, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newMaxCoverProblem(tt.args.atts, tt.args.k)
			if (err != nil) != tt.wantErr || !errors.Is(err, tt.expectedErr) {
				t.Errorf("newMaxCoverProblem() unexpected error, got: %v, want: %v", err, tt.expectedErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newMaxCoverProblem() got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestMaxCoverAttestationAggregation_maxCoverCandidateList_filter(t *testing.T) {
	type args struct {
		covered bitfield.Bitlist
	}
	var problem maxCoverCandidateList
	tests := []struct {
		name string
		cl   maxCoverCandidateList
		args args
		want *maxCoverCandidateList
	}{
		{
			name: "nil list",
			cl:   nil,
			args: args{},
			want: &problem,
		},
		{
			name: "empty list",
			cl:   maxCoverCandidateList{},
			args: args{},
			want: &maxCoverCandidateList{},
		},
		{
			name: "all processed",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, true},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
			},
			args: args{},
			want: &maxCoverCandidateList{},
		},
		{
			name: "partially processed",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
			},
			args: args{
				covered: bitfield.NewBitlist(8),
			},
			want: &maxCoverCandidateList{
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
			},
		},
		{
			name: "all overlapping",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
			},
			args: args{
				covered: bitlistWithAllBitsSet(8),
			},
			want: &maxCoverCandidateList{},
		},
		{
			name: "partially overlapping",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b01000011, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
			},
			args: args{
				covered: bitfield.Bitlist{0b10000001, 0b1},
			},
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
			},
		},
		{
			name: "overlapping and processed and pending",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000011, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
			},
			args: args{
				covered: bitfield.Bitlist{0b10000001, 0b1},
			},
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cl.filter(tt.args.covered)
			sort.Slice(*got, func(i, j int) bool {
				return (*got)[i].key < (*got)[j].key
			})
			sort.Slice(*tt.want, func(i, j int) bool {
				return (*tt.want)[i].key < (*tt.want)[j].key
			})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filter() unexpected result, got: %v, want: %v", got, tt.want)
			}
		})
	}
}
