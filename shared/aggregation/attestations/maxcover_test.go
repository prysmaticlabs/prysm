package attestations

import (
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestAggregateAttestations_MaxCover_NewMaxCover(t *testing.T) {
	type args struct {
		atts []*ethpb.Attestation
	}
	tests := []struct {
		name      string
		args      args
		want      *aggregation.MaxCoverProblem
		wantedErr string
	}{
		{
			name: "nil attestations",
			args: args{
				atts: nil,
			},
			wantedErr: ErrInvalidAttestationCount.Error(),
		},
		{
			name: "no attestations",
			args: args{
				atts: []*ethpb.Attestation{},
			},
			wantedErr: ErrInvalidAttestationCount.Error(),
		},
		{
			name: "attestations of different bitlist length",
			args: args{
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.NewBitlist(64)},
					{AggregationBits: bitfield.NewBitlist(128)},
				},
			},
			wantedErr: aggregation.ErrBitsDifferentLen.Error(),
		},
		{
			name: "single attestation",
			args: args{
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.Bitlist{0b00001010, 0b1}},
				},
			},
			want: &aggregation.MaxCoverProblem{
				Candidates: aggregation.MaxCoverCandidates{
					aggregation.NewMaxCoverCandidate(0, &bitfield.Bitlist{0b00001010, 0b1}),
				},
			},
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
			want: &aggregation.MaxCoverProblem{
				Candidates: aggregation.MaxCoverCandidates{
					aggregation.NewMaxCoverCandidate(0, &bitfield.Bitlist{0b00001010, 0b1}),
					aggregation.NewMaxCoverCandidate(1, &bitfield.Bitlist{0b00101010, 0b1}),
					aggregation.NewMaxCoverCandidate(2, &bitfield.Bitlist{0b11111010, 0b1}),
					aggregation.NewMaxCoverCandidate(3, &bitfield.Bitlist{0b00000010, 0b1}),
					aggregation.NewMaxCoverCandidate(4, &bitfield.Bitlist{0b00000001, 0b1}),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMaxCover(tt.args.atts)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
				assert.DeepEqual(t, tt.want, got)
			}
		})
	}
}

func TestAggregateAttestations_MaxCover_AttList_dedup(t *testing.T) {
	tests := []struct {
		name string
		atts attList
		want attList
	}{
		{
			name: "nil list",
			atts: nil,
			want: attList(nil),
		},
		{
			name: "empty list",
			atts: attList{},
			want: attList{},
		},
		{
			name: "single item",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{}},
			},
		},
		{
			name: "two items no duplicates",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10111110, 0x01}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01111111, 0x01}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01111111, 0x01}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10111110, 0x01}},
			},
		},
		{
			name: "two items with duplicates",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0xba, 0x01}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0xba, 0x01}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0xba, 0x01}},
			},
		},
		{
			name: "sorted no duplicates",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00101011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100000, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00101011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100000, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
			},
		},
		{
			name: "sorted with duplicates",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
		},
		{
			name: "all equal",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
			},
		},
		{
			name: "unsorted no duplicates",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00100010, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00100010, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
			},
		},
		{
			name: "unsorted with duplicates",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
			},
		},
		{
			name: "proper subset",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
			want: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atts := tt.atts.dedup()
			sort.Slice(atts, func(i, j int) bool {
				return atts[i].AggregationBits.Count() > atts[j].AggregationBits.Count()
			})
			assert.DeepEqual(t, tt.want, atts)
		})
	}
}
