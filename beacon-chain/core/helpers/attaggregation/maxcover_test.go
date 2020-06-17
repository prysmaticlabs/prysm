package attaggregation

import (
	"errors"
	"reflect"
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
					&maxCoverCandidate{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
					&maxCoverCandidate{1, &bitfield.Bitlist{0b00101010, 0b1}, 3, false},
					&maxCoverCandidate{2, &bitfield.Bitlist{0b11111010, 0b1}, 6, false},
					&maxCoverCandidate{3, &bitfield.Bitlist{0b00000010, 0b1}, 1, false},
					&maxCoverCandidate{4, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
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
