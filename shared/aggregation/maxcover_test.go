package aggregation

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	aggtesting "github.com/prysmaticlabs/prysm/shared/aggregation/testing"
)

func TestMaxCover_MaxCoverCandidates_filter(t *testing.T) {
	type args struct {
		covered       bitfield.Bitlist
		allowOverlaps bool
	}
	var problem MaxCoverCandidates
	tests := []struct {
		name string
		cl   MaxCoverCandidates
		args args
		want *MaxCoverCandidates
	}{
		{
			name: "nil list",
			cl:   nil,
			args: args{},
			want: &problem,
		},
		{
			name: "empty list",
			cl:   MaxCoverCandidates{},
			args: args{},
			want: &MaxCoverCandidates{},
		},
		{
			name: "all processed",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, true},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
			},
			args: args{},
			want: &MaxCoverCandidates{},
		},
		{
			name: "partially processed",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
			},
			args: args{
				covered: bitfield.NewBitlist(8),
			},
			want: &MaxCoverCandidates{
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
			},
		},
		{
			name: "all overlapping",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b01000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
			},
			args: args{
				covered: aggtesting.BitlistWithAllBitsSet(t, 8),
			},
			want: &MaxCoverCandidates{},
		},
		{
			name: "partially overlapping",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b01000011, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
			},
			args: args{
				covered: bitfield.Bitlist{0b10000001, 0b1},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
			},
		},
		{
			name: "overlapping and processed and pending",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{3, &bitfield.Bitlist{0b00001010, 0b1}, 2, true},
				{4, &bitfield.Bitlist{0b01000011, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
			},
			args: args{
				covered: bitfield.Bitlist{0b10000001, 0b1},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
			},
		},
		{
			name: "overlapping and processed and pending - allow overlaps",
			cl: MaxCoverCandidates{
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
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00001010, 0b1}, 2, false},
				{2, &bitfield.Bitlist{0b11000010, 0b1}, 2, false},
				{4, &bitfield.Bitlist{0b10001010, 0b1}, 2, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cl.filter(tt.args.covered, tt.args.allowOverlaps)
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

func TestMaxCover_MaxCoverCandidates_sort(t *testing.T) {
	var problem MaxCoverCandidates
	tests := []struct {
		name string
		cl   MaxCoverCandidates
		want *MaxCoverCandidates
	}{
		{
			name: "nil list",
			cl:   nil,
			want: &problem,
		},
		{
			name: "empty list",
			cl:   MaxCoverCandidates{},
			want: &MaxCoverCandidates{},
		},
		{
			name: "single item",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 5, false},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 5, false},
			},
		},
		{
			name: "already sorted",
			cl: MaxCoverCandidates{
				{5, &bitfield.Bitlist{}, 5, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{2, &bitfield.Bitlist{}, 2, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
			want: &MaxCoverCandidates{
				{5, &bitfield.Bitlist{}, 5, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{2, &bitfield.Bitlist{}, 2, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
		},
		{
			name: "all equal",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
				{0, &bitfield.Bitlist{}, 5, false},
			},
		},
		{
			name: "unsorted",
			cl: MaxCoverCandidates{
				{2, &bitfield.Bitlist{}, 2, false},
				{4, &bitfield.Bitlist{}, 4, false},
				{3, &bitfield.Bitlist{}, 4, false},
				{5, &bitfield.Bitlist{}, 5, false},
				{1, &bitfield.Bitlist{}, 1, false},
			},
			want: &MaxCoverCandidates{
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

func TestMaxCover_MaxCoverCandidates_union(t *testing.T) {
	tests := []struct {
		name string
		cl   MaxCoverCandidates
		want bitfield.Bitlist
	}{
		{
			name: "nil",
			cl:   nil,
			want: bitfield.Bitlist(nil),
		},
		{
			name: "single empty candidate",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000000, 0b1}, 0, false},
			},
			want: bitfield.Bitlist{0b00000000, 0b1},
		},
		{
			name: "single full candidate",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b11111111, 0b1}, 8, false},
			},
			want: aggtesting.BitlistWithAllBitsSet(t, 8),
		},
		{
			name: "mixed",
			cl: MaxCoverCandidates{
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

func TestMaxCover_MaxCoverCandidates_score(t *testing.T) {
	var problem MaxCoverCandidates
	tests := []struct {
		name      string
		cl        MaxCoverCandidates
		uncovered bitfield.Bitlist
		want      *MaxCoverCandidates
	}{
		{
			name: "nil",
			cl:   nil,
			want: &problem,
		},
		{
			name: "uncovered set is empty",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 1, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 3, false},
			},
			uncovered: bitfield.NewBitlist(8),
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
			},
		},
		{
			name: "completely uncovered",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
			},
			uncovered: aggtesting.BitlistWithAllBitsSet(t, 8),
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 1, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00011011, 0b1}, 4, false},
				{3, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 3, false},
			},
		},
		{
			name: "partial uncovered set",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00011011, 0b1}, 1, false},
				{2, &bitfield.Bitlist{0b10011011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b11111111, 0b1}, 1, false},
				{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
			},
			uncovered: bitfield.Bitlist{0b11010010, 0b1},
			want: &MaxCoverCandidates{
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

func TestMaxCover_MaxCoverCandidates_dedup(t *testing.T) {
	var problem MaxCoverCandidates
	tests := []struct {
		name string
		cl   MaxCoverCandidates
		want *MaxCoverCandidates
	}{
		{
			name: "nil list",
			cl:   nil,
			want: &problem,
		},
		{
			name: "empty list",
			cl:   MaxCoverCandidates{},
			want: &MaxCoverCandidates{},
		},
		{
			name: "single item",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 5, false},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 5, false},
			},
		},
		{
			name: "two items nil bitlists",
			cl: MaxCoverCandidates{
				{0, nil, 0, false},
				{0, nil, 0, false},
			},
			want: &MaxCoverCandidates{
				{0, nil, 0, false},
				{0, nil, 0, false},
			},
		},
		{
			name: "two items empty bitlists",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 0, false},
				{0, &bitfield.Bitlist{}, 0, false},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{}, 0, false},
				{0, &bitfield.Bitlist{}, 0, false},
			},
		},
		{
			name: "two items no duplicates",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0xfe, 0x01}, 0, false},
				{0, &bitfield.Bitlist{0xff, 0x01}, 0, false},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0xff, 0x01}, 8, false},
				{0, &bitfield.Bitlist{0xfe, 0x01}, 7, false},
			},
		},
		{
			name: "two items with duplicates",
			cl: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0xba, 0x01}, 0, false},
				{0, &bitfield.Bitlist{0xba, 0x01}, 0, false},
			},
			want: &MaxCoverCandidates{
				{0, &bitfield.Bitlist{0xba, 0x01}, 5, false},
			},
		},
		{
			name: "sorted no duplicates",
			cl: MaxCoverCandidates{
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
			},
			want: &MaxCoverCandidates{
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 6, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 5, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 2, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
			},
		},
		{
			name: "sorted with duplicates",
			cl: MaxCoverCandidates{
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
			},
			want: &MaxCoverCandidates{
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 6, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 5, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 2, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
			},
		},
		{
			name: "all equal",
			cl: MaxCoverCandidates{
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
			},
			want: &MaxCoverCandidates{
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 2, false},
			},
		},
		{
			name: "unsorted no duplicates",
			cl: MaxCoverCandidates{
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 0, false},
			},
			want: &MaxCoverCandidates{
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 6, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 5, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 2, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
			},
		},
		{
			name: "unsorted with duplicates",
			cl: MaxCoverCandidates{
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 0, false},
			},
			want: &MaxCoverCandidates{
				{3, &bitfield.Bitlist{0b11001111, 0b1}, 6, false},
				{5, &bitfield.Bitlist{0b01101101, 0b1}, 5, false},
				{4, &bitfield.Bitlist{0b00001111, 0b1}, 4, false},
				{2, &bitfield.Bitlist{0b00000011, 0b1}, 2, false},
				{1, &bitfield.Bitlist{0b00000001, 0b1}, 1, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cl.dedup(false)
			sort.Slice(*got, func(i, j int) bool {
				if (*got)[i].score == (*got)[j].score {
					return (*got)[i].key < (*got)[j].key
				}
				return (*got)[i].score > (*got)[j].score
			})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dedup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxCover_MaxCoverProblem_cover(t *testing.T) {
	problemSet := func() MaxCoverCandidates {
		// test vectors originally from:
		// https://github.com/sigp/lighthouse/blob/master/beacon_node/operation_pool/src/max_cover.rs
		return MaxCoverCandidates{
			{0, &bitfield.Bitlist{0b00000100, 0b1}, 0, false},
			{1, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
			{2, &bitfield.Bitlist{0b00011011, 0b1}, 0, false},
			{3, &bitfield.Bitlist{0b00000001, 0b1}, 0, false},
			{4, &bitfield.Bitlist{0b00011010, 0b1}, 0, false},
		}
	}
	type args struct {
		k               int
		candidates      MaxCoverCandidates
		allowOverlaps   bool
		allowDuplicates bool
	}
	tests := []struct {
		name        string
		args        args
		want        *Aggregation
		wantErr     bool
		expectedErr error
	}{
		{
			name:        "nil problem",
			args:        args{},
			wantErr:     true,
			expectedErr: ErrInvalidMaxCoverProblem,
		},
		{
			name: "different bitlengths",
			args: args{k: 3, candidates: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000000, 0b00011111, 0xf1}, 0, false},
				{2, &bitfield.Bitlist{0b00000001, 0b11100000, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000110, 0b00000000, 0b1}, 0, false},
			}},
			wantErr:     true,
			expectedErr: ErrInvalidMaxCoverProblem,
		},
		{
			name: "k=0",
			args: args{k: 0, candidates: problemSet(), allowDuplicates: true},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0000000, 0b1},
				Keys:     []int{},
			},
			wantErr: false,
		},
		{
			name: "k=1",
			args: args{k: 1, candidates: problemSet(), allowDuplicates: true},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011011, 0b1},
				Keys:     []int{1},
			},
			wantErr: false,
		},
		{
			name: "k=2",
			args: args{k: 2, candidates: problemSet(), allowDuplicates: true},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "k=3",
			args: args{k: 3, candidates: problemSet(), allowDuplicates: true},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "k=5",
			args: args{k: 5, candidates: problemSet(), allowDuplicates: true},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "k=50",
			args: args{k: 50, candidates: problemSet(), allowDuplicates: true},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
			wantErr: false,
		},
		{
			name: "suboptimal", // Greedy algorithm selects: 0, 2, 3, while 1,4,5 is optimal.
			args: args{k: 3, candidates: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000000, 0b00011111, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000001, 0b11100000, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000110, 0b00000000, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00110000, 0b01110000, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00000110, 0b10001100, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b01001001, 0b00000011, 0b1}, 0, false},
			}},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b00000111, 0b11111111, 0b1},
				Keys:     []int{0, 2, 3},
			},
			wantErr: false,
		},
		{
			name: "allow overlaps",
			args: args{k: 5, allowOverlaps: true, candidates: MaxCoverCandidates{
				{0, &bitfield.Bitlist{0b00000000, 0b00000001, 0b11111110, 0b1}, 0, false},
				{1, &bitfield.Bitlist{0b00000000, 0b00001110, 0b00001110, 0b1}, 0, false},
				{2, &bitfield.Bitlist{0b00000000, 0b01110000, 0b01110000, 0b1}, 0, false},
				{3, &bitfield.Bitlist{0b00000111, 0b10000001, 0b10000000, 0b1}, 0, false},
				{4, &bitfield.Bitlist{0b00000000, 0b00000110, 0b00000110, 0b1}, 0, false},
				{5, &bitfield.Bitlist{0b00000000, 0b00000001, 0b01100010, 0b1}, 0, false},
				{6, &bitfield.Bitlist{0b00001000, 0b00001000, 0b10000010, 0b1}, 0, false},
			}},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b00001111, 0xff, 0b11111110, 0b1},
				Keys:     []int{0, 3, 1, 2, 6},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &MaxCoverProblem{
				Candidates: tt.args.candidates,
			}
			got, err := mc.Cover(tt.args.k, tt.args.allowOverlaps, tt.args.allowDuplicates)
			if (err != nil) != tt.wantErr || !errors.Is(err, tt.expectedErr) {
				t.Errorf("newMaxCoverProblem() unexpected error, got: %v, want: %v", err, tt.expectedErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cover() got: %v, want: %v", got, tt.want)
			}
		})
	}
}
