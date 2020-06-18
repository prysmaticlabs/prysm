package aggregation

import (
	"errors"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
)

func TestAttestations_NewAttestationsMaxCover(t *testing.T) {
	type args struct {
		atts []*ethpb.Attestation
	}
	tests := []struct {
		name        string
		args        args
		want        *MaxCoverProblem
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
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.Bitlist{0b00001010, 0b1}},
				},
			},
			want: &MaxCoverProblem{
				Candidates: MaxCoverCandidates{
					{0, &bitfield.Bitlist{0b00001010, 0b1}, 0, false},
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
			},
			want: &MaxCoverProblem{
				Candidates: MaxCoverCandidates{
					{0, &bitfield.Bitlist{0b00001010, 0b1}, 0, false},
					{1, &bitfield.Bitlist{0b00101010, 0b1}, 0, false},
					{2, &bitfield.Bitlist{0b11111010, 0b1}, 0, false},
					{3, &bitfield.Bitlist{0b00000010, 0b1}, 0, false},
					{4, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAttestationsMaxCover(tt.args.atts)
			if (err != nil) != tt.wantErr || !errors.Is(err, tt.expectedErr) {
				t.Errorf("NewMaxCoverProblem() unexpected error, got: %v, want: %v", err, tt.expectedErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMaxCoverProblem() got: %v, want: %v", got, tt.want)
			}
		})
	}
}
