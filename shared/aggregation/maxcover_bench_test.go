package aggregation

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	aggtesting "github.com/prysmaticlabs/prysm/shared/aggregation/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func BenchmarkMaxCoverProblem_Cover(b *testing.B) {
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
		k             int
		candidates    MaxCoverCandidates
		allowOverlaps bool
	}
	tests := []struct {
		name      string
		args      args
		want      *Aggregation
		wantedErr string
	}{
		{
			name:      "nil problem",
			args:      args{},
			wantedErr: ErrInvalidMaxCoverProblem.Error(),
		},
		{
			name: "k=0",
			args: args{k: 0, candidates: problemSet()},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0000000, 0b1},
				Keys:     []int{},
			},
		},
		{
			name: "k=1",
			args: args{k: 1, candidates: problemSet()},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011011, 0b1},
				Keys:     []int{1},
			},
		},
		{
			name: "k=2",
			args: args{k: 2, candidates: problemSet()},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
		},
		{
			name: "k=3",
			args: args{k: 3, candidates: problemSet()},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
		},
		{
			name: "k=5",
			args: args{k: 5, candidates: problemSet()},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
		},
		{
			name: "k=50",
			args: args{k: 50, candidates: problemSet()},
			want: &Aggregation{
				Coverage: bitfield.Bitlist{0b0011111, 0b1},
				Keys:     []int{1, 0},
			},
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
		},
	}
	copyCandidates := func(candidates MaxCoverCandidates) MaxCoverCandidates {
		copyCandidates := make(MaxCoverCandidates, 0, len(candidates))
		for _, candidate := range candidates {
			copyCandidates = append(copyCandidates, NewMaxCoverCandidate(candidate.key, candidate.bits))
		}
		return copyCandidates
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				mc := &MaxCoverProblem{
					Candidates: copyCandidates(tt.args.candidates),
				}
				got, err := mc.Cover(tt.args.k, tt.args.allowOverlaps)
				if tt.wantedErr != "" {
					require.ErrorContains(b, tt.wantedErr, err)
				} else {
					require.NoError(b, err)
					require.DeepEqual(b, tt.want, got)
				}
			}
		})
	}
}

func BenchmarkMaxCoverProblem_MaxCover(b *testing.B) {
	bitlistLen := params.BeaconConfig().MaxValidatorsPerCommittee
	tests := []struct {
		name          string
		numCandidates uint64
		numMarkedBits uint64
		allowOverlaps bool
	}{
		{
			name:          "128_attestations_with_single_bit_set",
			numCandidates: 128,
			numMarkedBits: 8,
		},
		{
			name:          "1024_attestations_with_single_bit_set",
			numCandidates: 1024,
			numMarkedBits: 8,
		},
		{
			name:          "2048_attestations_with_single_bit_set",
			numCandidates: 2048,
			numMarkedBits: 8,
		},
	}
	for _, tt := range tests {
		bitlists := aggtesting.Bitlists64WithSingleBitSet(tt.numCandidates, bitlistLen)
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := MaxCover(bitlists, len(bitlists), tt.allowOverlaps)
				_ = err
			}
		})
	}
}
