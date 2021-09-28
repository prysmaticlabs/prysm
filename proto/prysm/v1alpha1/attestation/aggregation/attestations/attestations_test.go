package attestations

import (
	"fmt"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation/aggregation"
	aggtesting "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation/aggregation/testing"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	resetCfg := features.InitWithReset(&features.Flags{
		AttestationAggregationStrategy: string(OptMaxCoverAggregation),
	})
	defer resetCfg()
	m.Run()
}

func TestAggregateAttestations_AggregatePair(t *testing.T) {
	tests := []struct {
		a1   *ethpb.Attestation
		a2   *ethpb.Attestation
		want *ethpb.Attestation
	}{
		{
			a1:   &ethpb.Attestation{AggregationBits: []byte{}},
			a2:   &ethpb.Attestation{AggregationBits: []byte{}},
			want: &ethpb.Attestation{AggregationBits: []byte{}},
		},
		{
			a1:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x03}},
			a2:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x02}},
			want: &ethpb.Attestation{AggregationBits: []byte{0x03}},
		},
		{
			a1:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x02}},
			a2:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x03}},
			want: &ethpb.Attestation{AggregationBits: []byte{0x03}},
		},
	}
	for _, tt := range tests {
		got, err := AggregatePair(tt.a1, tt.a2)
		require.NoError(t, err)
		require.Equal(t, true, ssz.DeepEqual(got, tt.want))
	}
}

func TestAggregateAttestations_AggregatePair_OverlapFails(t *testing.T) {
	tests := []struct {
		a1 *ethpb.Attestation
		a2 *ethpb.Attestation
	}{
		{
			a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x1F}},
			a2: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x11}},
		},
		{
			a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0xFF, 0x85}},
			a2: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x13, 0x8F}},
		},
	}
	for _, tt := range tests {
		_, err := AggregatePair(tt.a1, tt.a2)
		require.ErrorContains(t, aggregation.ErrBitsOverlap.Error(), err)
	}
}

func TestAggregateAttestations_AggregatePair_DiffLengthFails(t *testing.T) {
	tests := []struct {
		a1 *ethpb.Attestation
		a2 *ethpb.Attestation
	}{
		{
			a1: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x0F}},
			a2: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x11}},
		},
	}
	for _, tt := range tests {
		_, err := AggregatePair(tt.a1, tt.a2)
		require.ErrorContains(t, bitfield.ErrBitlistDifferentLength.Error(), err)
	}
}

func TestAggregateAttestations_Aggregate(t *testing.T) {
	// Each test defines the aggregation bitfield inputs and the wanted output result.
	bitlistLen := params.BeaconConfig().MaxValidatorsPerCommittee
	tests := []struct {
		name   string
		inputs []bitfield.Bitlist
		want   []bitfield.Bitlist
		err    error
	}{
		{
			name:   "empty list",
			inputs: []bitfield.Bitlist{},
			want:   []bitfield.Bitlist{},
		},
		{
			name: "single attestation",
			inputs: []bitfield.Bitlist{
				{0b00000010, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000010, 0b1},
			},
		},
		{
			name: "two attestations with no overlap",
			inputs: []bitfield.Bitlist{
				{0b00000001, 0b1},
				{0b00000010, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b1},
			},
		},
		{
			name:   "256 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(256, bitlistLen),
			want: []bitfield.Bitlist{
				aggtesting.BitlistWithAllBitsSet(256),
			},
		},
		{
			name:   "1024 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(1024, bitlistLen),
			want: []bitfield.Bitlist{
				aggtesting.BitlistWithAllBitsSet(1024),
			},
		},
		{
			name: "two attestations with overlap",
			inputs: []bitfield.Bitlist{
				{0b00000101, 0b1},
				{0b00000110, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000101, 0b1},
				{0b00000110, 0b1},
			},
		},
		{
			name: "some attestations overlap",
			inputs: []bitfield.Bitlist{
				{0b00001001, 0b1},
				{0b00010110, 0b1},
				{0b00001010, 0b1},
				{0b00110001, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00111011, 0b1},
				{0b00011111, 0b1},
			},
		},
		{
			name: "some attestations produce duplicates which are removed",
			inputs: []bitfield.Bitlist{
				{0b00000101, 0b1},
				{0b00000110, 0b1},
				{0b00001010, 0b1},
				{0b00001001, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00001111, 0b1}, // both 0&1 and 2&3 produce this bitlist
			},
		},
		{
			name: "two attestations where one is fully contained within the other",
			inputs: []bitfield.Bitlist{
				{0b00000001, 0b1},
				{0b00000011, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b1},
			},
		},
		{
			name: "two attestations where one is fully contained within the other reversed",
			inputs: []bitfield.Bitlist{
				{0b00000011, 0b1},
				{0b00000001, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b1},
			},
		},
		{
			name: "attestations with different bitlist lengths",
			inputs: []bitfield.Bitlist{
				{0b00000011, 0b10},
				{0b00000111, 0b100},
				{0b00000100, 0b1},
			},
			want: []bitfield.Bitlist{
				{0b00000011, 0b10},
				{0b00000111, 0b100},
				{0b00000100, 0b1},
			},
			err: bitfield.ErrBitlistDifferentLength,
		},
	}

	for _, tt := range tests {
		runner := func() {
			got, err := Aggregate(aggtesting.MakeAttestationsFromBitlists(tt.inputs))
			if tt.err != nil {
				require.ErrorContains(t, tt.err.Error(), err)
				return
			}
			require.NoError(t, err)
			sort.Slice(got, func(i, j int) bool {
				return got[i].AggregationBits.Bytes()[0] < got[j].AggregationBits.Bytes()[0]
			})
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Bytes()[0] < tt.want[j].Bytes()[0]
			})
			assert.Equal(t, len(tt.want), len(got))
			for i, w := range tt.want {
				assert.DeepEqual(t, w.Bytes(), got[i].AggregationBits.Bytes())
			}
		}
		t.Run(fmt.Sprintf("%s/%s", tt.name, NaiveAggregation), func(t *testing.T) {
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(NaiveAggregation),
			})
			defer resetCfg()
			runner()
		})
		t.Run(fmt.Sprintf("%s/%s", tt.name, MaxCoverAggregation), func(t *testing.T) {
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(MaxCoverAggregation),
			})
			defer resetCfg()
			runner()
		})
		t.Run(fmt.Sprintf("%s/%s", tt.name, OptMaxCoverAggregation), func(t *testing.T) {
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(OptMaxCoverAggregation),
			})
			defer resetCfg()
			runner()
		})
	}

	t.Run("invalid strategy", func(t *testing.T) {
		resetCfg := features.InitWithReset(&features.Flags{
			AttestationAggregationStrategy: "foobar",
		})
		defer resetCfg()
		_, err := Aggregate(aggtesting.MakeAttestationsFromBitlists([]bitfield.Bitlist{}))
		assert.ErrorContains(t, "\"foobar\": invalid aggregation strategy", err)
	})

	t.Run("broken attestation bitset", func(t *testing.T) {
		wantErr := "bitlist cannot be nil or empty: invalid max_cover problem"
		t.Run(string(MaxCoverAggregation), func(t *testing.T) {
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(MaxCoverAggregation),
			})
			defer resetCfg()
			_, err := Aggregate(aggtesting.MakeAttestationsFromBitlists([]bitfield.Bitlist{
				{0b00000011, 0b0},
				{0b00000111, 0b100},
				{0b00000100, 0b1},
			}))
			assert.ErrorContains(t, wantErr, err)
		})
		t.Run(string(OptMaxCoverAggregation), func(t *testing.T) {
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(OptMaxCoverAggregation),
			})
			defer resetCfg()
			_, err := Aggregate(aggtesting.MakeAttestationsFromBitlists([]bitfield.Bitlist{
				{0b00000011, 0b0},
				{0b00000111, 0b100},
				{0b00000100, 0b1},
			}))
			assert.ErrorContains(t, wantErr, err)
		})
	})

	t.Run("candidate swapping when aggregating", func(t *testing.T) {
		// The first item cannot be aggregated, and should be pushed down the list,
		// by two swaps with aggregated items (aggregation is done in-place, so the very same
		// underlying array is used for storing both aggregated and non-aggregated items).
		resetCfg := features.InitWithReset(&features.Flags{
			AttestationAggregationStrategy: string(OptMaxCoverAggregation),
		})
		defer resetCfg()
		got, err := Aggregate(aggtesting.MakeAttestationsFromBitlists([]bitfield.Bitlist{
			{0b10000000, 0b1},
			{0b11000101, 0b1},
			{0b00011000, 0b1},
			{0b01010100, 0b1},
			{0b10001010, 0b1},
		}))
		want := []bitfield.Bitlist{
			{0b11011101, 0b1},
			{0b11011110, 0b1},
			{0b10000000, 0b1},
		}
		assert.NoError(t, err)
		assert.Equal(t, len(want), len(got))
		for i, w := range want {
			assert.DeepEqual(t, w.Bytes(), got[i].AggregationBits.Bytes())
		}
	})
}

func TestAggregateAttestations_PerformanceComparison(t *testing.T) {
	// Tests below are examples of cases where max-cover's greedy approach outperforms the original
	// naive aggregation (which is very much dependent on order in which items are fed into it).
	tests := []struct {
		name     string
		bitsList [][]byte
	}{
		{
			name: "test1",
			bitsList: [][]byte{
				{0b00000100, 0b1},
				{0b00000010, 0b1},
				{0b00000001, 0b1},
				{0b00011001, 0b1},
			},
		},
		{
			name: "test2",
			bitsList: [][]byte{
				{0b10010001, 0b1},
				{0b00100000, 0b1},
				{0b01101110, 0b1},
			},
		},
		{
			name: "test3",
			bitsList: [][]byte{
				{0b00100000, 0b00000011, 0b1},
				{0b00011100, 0b11000000, 0b1},
				{0b11111100, 0b00000000, 0b1},
				{0b00000011, 0b10000000, 0b1},
				{0b11100011, 0b00000000, 0b1},
			},
		},
	}

	scoreAtts := func(atts []*ethpb.Attestation) uint64 {
		score := uint64(0)
		sort.Slice(atts, func(i, j int) bool {
			return atts[i].AggregationBits.Count() > atts[j].AggregationBits.Count()
		})
		// Score the best aggregate.
		if len(atts) > 0 {
			score = atts[0].AggregationBits.Count()
		}
		return score
	}

	generateAtts := func(bitsList [][]byte) []*ethpb.Attestation {
		sign := bls.NewAggregateSignature().Marshal()
		atts := make([]*ethpb.Attestation, 0)
		for _, b := range bitsList {
			atts = append(atts, &ethpb.Attestation{
				AggregationBits: b,
				Signature:       sign,
			})
		}
		return atts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atts, err := NaiveAttestationAggregation(generateAtts(tt.bitsList))
			require.NoError(t, err)
			score1 := scoreAtts(atts)

			atts, err = MaxCoverAttestationAggregation(generateAtts(tt.bitsList))
			require.NoError(t, err)
			score2 := scoreAtts(atts)

			t.Logf("native = %d, max-cover: %d\n", score1, score2)
			assert.Equal(t, true, score1 <= score2,
				"max-cover failed to produce higher score (naive: %d, max-cover: %d)", score1, score2)
		})
	}
}
