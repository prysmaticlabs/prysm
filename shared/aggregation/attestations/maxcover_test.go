package attestations

import (
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
		name string
		args args
		want *aggregation.MaxCoverProblem
	}{
		{
			name: "nil attestations",
			args: args{
				atts: nil,
			},
			want: &aggregation.MaxCoverProblem{Candidates: []*aggregation.MaxCoverCandidate{}},
		},
		{
			name: "no attestations",
			args: args{
				atts: []*ethpb.Attestation{},
			},
			want: &aggregation.MaxCoverProblem{Candidates: []*aggregation.MaxCoverCandidate{}},
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
			assert.DeepEqual(t, tt.want, NewMaxCover(tt.args.atts))
		})
	}
}

func TestAggregateAttestations_MaxCover_AttList_validate(t *testing.T) {
	tests := []struct {
		name      string
		atts      attList
		wantedErr string
	}{
		{
			name:      "nil list",
			atts:      nil,
			wantedErr: "nil list",
		},
		{
			name:      "empty list",
			atts:      attList{},
			wantedErr: "empty list",
		},
		{
			name:      "first bitlist is nil",
			atts:      attList{&ethpb.Attestation{}},
			wantedErr: "bitlist cannot be nil or empty",
		},
		{
			name: "non first bitlist is nil",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{},
			},
			wantedErr: aggregation.ErrBitsDifferentLen.Error(),
		},
		{
			name: "first bitlist is empty",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{}},
			},
			wantedErr: "bitlist cannot be nil or empty",
		},
		{
			name: "non first bitlist is empty",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{}},
			},
			wantedErr: aggregation.ErrBitsDifferentLen.Error(),
		},
		{
			name: "bitlists of non equal length",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(63)},
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
			},
			wantedErr: aggregation.ErrBitsDifferentLen.Error(),
		},
		{
			name: "valid bitlists",
			atts: attList{
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
				&ethpb.Attestation{AggregationBits: bitfield.NewBitlist(64)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.atts.validate()
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAggregateAttestations_rearrangeProcessedAttestations(t *testing.T) {
	tests := []struct {
		name     string
		atts     []*ethpb.Attestation
		keys     []int
		wantAtts []*ethpb.Attestation
	}{
		{
			name: "nil attestations",
		},
		{
			name: "single attestation no processed keys",
			atts: []*ethpb.Attestation{
				{},
			},
			wantAtts: []*ethpb.Attestation{
				{},
			},
		},
		{
			name: "single attestation processed",
			atts: []*ethpb.Attestation{
				{},
			},
			keys: []int{0},
			wantAtts: []*ethpb.Attestation{
				nil,
			},
		},
		{
			name: "multiple processed, last attestation marked",
			atts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x02}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				{AggregationBits: bitfield.Bitlist{0x04}},
			},
			keys: []int{1, 4}, // Only attestation at index 1, should be moved, att at 4 is already at the end.
			wantAtts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				{AggregationBits: bitfield.Bitlist{0x02}},
				nil, nil,
			},
		},
		{
			name: "all processed",
			atts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x02}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				{AggregationBits: bitfield.Bitlist{0x04}},
			},
			keys: []int{0, 1, 2, 3, 4},
			wantAtts: []*ethpb.Attestation{
				nil, nil, nil, nil, nil,
			},
		},
		{
			name: "operate on slice, single attestation marked",
			atts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x02}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				{AggregationBits: bitfield.Bitlist{0x04}},
				// Assuming some attestations have been already marked as nil, during previous rounds:
				nil, nil, nil,
			},
			keys: []int{2},
			wantAtts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x04}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				nil, nil, nil, nil,
			},
		},
		{
			name: "operate on slice, non-last attestation marked",
			atts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x02}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				{AggregationBits: bitfield.Bitlist{0x04}},
				{AggregationBits: bitfield.Bitlist{0x05}},
				// Assuming some attestations have been already marked as nil, during previous rounds:
				nil, nil, nil,
			},
			keys: []int{2, 3},
			wantAtts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x05}},
				{AggregationBits: bitfield.Bitlist{0x04}},
				nil, nil, nil, nil, nil,
			},
		},
		{
			name: "operate on slice, last attestation marked",
			atts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x02}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				{AggregationBits: bitfield.Bitlist{0x04}},
				// Assuming some attestations have been already marked as nil, during previous rounds:
				nil, nil, nil,
			},
			keys: []int{2, 4},
			wantAtts: []*ethpb.Attestation{
				{AggregationBits: bitfield.Bitlist{0x00}},
				{AggregationBits: bitfield.Bitlist{0x01}},
				{AggregationBits: bitfield.Bitlist{0x03}},
				nil, nil, nil, nil, nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := make([]*bitfield.Bitlist64, len(tt.atts))
			for i := 0; i < len(tt.atts); i++ {
				if tt.atts[i] != nil {
					candidates[i] = bitfield.NewBitlist64FromBytes(tt.atts[i].AggregationBits.Bytes())
				}
			}
			rearrangeProcessedAttestations(tt.atts, candidates, tt.keys)
			assert.DeepEqual(t, tt.atts, tt.wantAtts)
		})
	}
}
