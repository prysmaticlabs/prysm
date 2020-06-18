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
				atts: []*ethpb.Attestation{
					{AggregationBits: bitfield.Bitlist{0b00001010, 0b1}},
				},
			},
			want: &maxCoverProblem{
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
			},
			want: &maxCoverProblem{
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
			got, err := newMaxCoverProblem(tt.args.atts)
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
		covered       bitfield.Bitlist
		allowOverlaps bool
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
		{
			name: "overlapping and processed and pending - allow overlaps",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000011, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
			},
			args: args{
				covered:       bitfield.Bitlist{0b11111111, 0b1},
				allowOverlaps: true,
			},
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
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

func TestMaxCoverAttestationAggregation_maxCoverCandidateList_sort(t *testing.T) {
	var problem maxCoverCandidateList
	tests := []struct {
		name string
		cl   maxCoverCandidateList
		want *maxCoverCandidateList
	}{
		{
			name: "nil list",
			cl:   nil,
			want: &problem,
		},
		{
			name: "empty list",
			cl:   maxCoverCandidateList{},
			want: &maxCoverCandidateList{},
		},
		{
			name: "single item",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{}, 5, false},
			},
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{}, 5, false},
			},
		},
		{
			name: "already sorted",
			cl: maxCoverCandidateList{
				{5, &bitfield.Bitlist{}, 5, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{2, &bitfield.Bitlist{}, 2, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
			want: &maxCoverCandidateList{
				{5, &bitfield.Bitlist{}, 5, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{2, &bitfield.Bitlist{}, 2, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
		},
		{
			name: "all equal",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
			},
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
			},
		},
		{
			name: "unsorted",
			cl: maxCoverCandidateList{
				{2, &bitfield.Bitlist{}, 2, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{5, &bitfield.Bitlist{}, 5, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
			want: &maxCoverCandidateList{
				{5, &bitfield.Bitlist{}, 5, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{2, &bitfield.Bitlist{}, 2, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cl.sort(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxCoverAttestationAggregation_maxCoverCandidateList_union(t *testing.T) {
	tests := []struct {
		name string
		cl   maxCoverCandidateList
		want bitfield.Bitlist
	}{
		{
			name: "nil",
			cl:   nil,
			want: bitfield.Bitlist(nil),
		},
		{
			name: "single empty candidate",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000000, 0b1}, 0, false},
			},
			want: bitfield.Bitlist{0b00000000, 0b1},
		},
		{
			name: "single full candidate",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b11111111, 0b1}, 8, false},
			},
			want: bitlistWithAllBitsSet(8),
		},
		{
			name: "mixed",
			cl: maxCoverCandidateList{
				{1, &bitfield.Bitlist{0b00000000, 0b00001110, 0b00001110, 0b1}, 6, false},
				{2, &bitfield.Bitlist{0b00000000, 0b01110000, 0b01110000, 0b1}, 6, false},
				{3, &bitfield.Bitlist{0b00000111, 0b10000001, 0b10000000, 0b1}, 6, false},
				{4, &bitfield.Bitlist{0b00000000, 0b00000110, 0b00000110, 0b1}, 4, false},
				{5, &bitfield.Bitlist{0b10000000, 0b00000001, 0b01100010, 0b1}, 4, false},
				{6, &bitfield.Bitlist{0b00001000, 0b00001000, 0b10000010, 0b1}, 4, false},
				{7, &bitfield.Bitlist{0b00000000, 0b00000001, 0b11111110, 0b1}, 8, false},
			},
			want: bitfield.Bitlist{0b10001111, 0b11111111, 0b11111110, 0b1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cl.union(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("union(), got: %#b, want: %#b", got, tt.want)
			}
		})
	}
}

func TestMaxCoverAttestationAggregation_maxCoverCandidateList_score(t *testing.T) {
	var problem maxCoverCandidateList
	tests := []struct {
		name      string
		cl        maxCoverCandidateList
		uncovered bitfield.Bitlist
		want      *maxCoverCandidateList
	}{
		{
			name: "nil",
			cl:   nil,
			want: &problem,
		},
		{
			name: "uncovered set is empty",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 1, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 3, false},
			},
			uncovered: bitfield.NewBitlist(8),
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
			},
		},
		{
			name: "completely uncovered",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
			},
			uncovered: bitlistWithAllBitsSet(8),
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 1, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 3, false},
			},
		},
		{
			name: "partial uncovered set",
			cl: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 1, false},
				{2, &bitfield.Bitlist{0b10011011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b11111111, 0b1}, 1, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
			},
			uncovered: bitfield.Bitlist{0b11010010, 0b1},
			want: &maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b10011011, 0b1}, 3, false},
				{3, &bitfield.Bitlist{0b11111111, 0b1}, 4, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 2, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cl.score(tt.uncovered); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("score() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxCoverAttestationAggregation_maxCoverProblem_cover(t *testing.T) {
	problemSet := func() maxCoverCandidateList {
		// test vectors originally from:
		// https://github.com/sigp/lighthouse/blob/master/beacon_node/operation_pool/src/max_cover.rs
		return maxCoverCandidateList{
			{0, &bitfield.Bitlist{0b00000100, 0b1}, 1, false},
			{1, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
			{2, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
			{3, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
			{4, &bitfield.Bitlist{0b00011010, 0b1}, 3, false},
		}
	}
	type args struct {
		k             int
		candidates    maxCoverCandidateList
		allowOverlaps bool
	}
	tests := []struct {
		name        string
		args        args
		want        *maxCoverSolution
		wantErr     bool
		expectedErr error
	}{
		{
			name:        "nil problem",
			args:        args{},
			want:        nil,
			wantErr:     true,
			expectedErr: ErrInvalidMaxCoverProblem,
		},
		{
			name: "k=0",
			args: args{k: 0, candidates: problemSet()},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b0000000, 0b1},
				keys:     []int{},
			},
			wantErr: false,
		},
		{
			name: "k=1",
			args: args{k: 1, candidates: problemSet()},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b0011011, 0b1},
				keys:     []int{1},
			},
			wantErr: false,
		},
		{
			name: "k=2",
			args: args{k: 2, candidates: problemSet()},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b0011111, 0b1},
				keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "k=3",
			args: args{k: 3, candidates: problemSet()},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b0011111, 0b1},
				keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "k=5",
			args: args{k: 5, candidates: problemSet()},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b0011111, 0b1},
				keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "k=50",
			args: args{k: 50, candidates: problemSet()},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b0011111, 0b1},
				keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "suboptimal", // Greedy algorithm selects: 0, 2, 3, while 1,4,5 is optimal.
			args: args{k: 3, candidates: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000000, 0b00011111, 0b1}, 5, false},
				{2, &bitfield.Bitlist{0b00000001, 0b11100000, 0b1}, 4, false},
				{3, &bitfield.Bitlist{0b00000110, 0b00000000, 0b1}, 2, false},
				{1, &bitfield.Bitlist{0b00110000, 0b01110000, 0b1}, 5, false},
				{4, &bitfield.Bitlist{0b00000110, 0b10001100, 0b1}, 5, false},
				{5, &bitfield.Bitlist{0b01001001, 0b00000011, 0b1}, 5, false},
			}},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b00000111, 0b11111111, 0b1},
				keys:     []int{0, 2, 3},
			},
			wantErr: false,
		},
		{
			name: "prevent overlaps",
			args: args{k: 5, candidates: maxCoverCandidateList{
				{0, &bitfield.Bitlist{0b00000000, 0b00000001, 0b11111110, 0b1}, 8, false},
				{1, &bitfield.Bitlist{0b00000000, 0b00001110, 0b00001110, 0b1}, 6, false},
				{2, &bitfield.Bitlist{0b00000000, 0b01110000, 0b01110000, 0b1}, 6, false},
				{3, &bitfield.Bitlist{0b00000111, 0b10000001, 0b10000000, 0b1}, 6, false},
				{4, &bitfield.Bitlist{0b00000000, 0b00000110, 0b00000110, 0b1}, 4, false},
				{5, &bitfield.Bitlist{0b00000000, 0b00000001, 0b01100010, 0b1}, 4, false},
				{6, &bitfield.Bitlist{0b00001000, 0b00001000, 0b10000010, 0b1}, 4, false},
			}},
			want: &maxCoverSolution{
				coverage: bitfield.Bitlist{0b00000111, 0b11111111, 0b1},
				keys:     []int{0},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &maxCoverProblem{
				candidates: tt.args.candidates,
			}
			got, err := mc.cover(tt.args.k, tt.args.allowOverlaps)
			if (err != nil) != tt.wantErr || !errors.Is(err, tt.expectedErr) {
				t.Errorf("newMaxCoverProblem() unexpected error, got: %v, want: %v", err, tt.expectedErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cover() got: %v, want: %v", got, tt.want)
			}
		})
	}
}
