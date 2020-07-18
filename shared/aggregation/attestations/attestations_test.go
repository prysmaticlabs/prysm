package attestations

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
	aggtesting "github.com/prysmaticlabs/prysm/shared/aggregation/testing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
		AttestationAggregationStrategy: string(MaxCoverAggregation),
	})
	defer resetCfg()
	os.Exit(m.Run())
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
		require.ErrorContains(t, aggregation.ErrBitsDifferentLen.Error(), err)
	}
}

func TestAggregateAttestations_Aggregate(t *testing.T) {
	// Each test defines the aggregation bitfield inputs and the wanted output result.
	bitlistLen := params.BeaconConfig().MaxValidatorsPerCommittee
	tests := []struct {
		name   string
		inputs []bitfield.Bitlist
		want   []bitfield.Bitlist
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
			inputs: aggtesting.BitlistsWithSingleBitSet(t, 256, bitlistLen),
			want: []bitfield.Bitlist{
				aggtesting.BitlistWithAllBitsSet(t, 256),
			},
		},
		{
			name:   "1024 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(t, 1024, bitlistLen),
			want: []bitfield.Bitlist{
				aggtesting.BitlistWithAllBitsSet(t, 1024),
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
		},
	}

	for _, tt := range tests {
		runner := func() {
			got, err := Aggregate(aggtesting.MakeAttestationsFromBitlists(t, tt.inputs))
			require.NoError(t, err)
			sort.Slice(got, func(i, j int) bool {
				return got[i].AggregationBits.Bytes()[0] < got[j].AggregationBits.Bytes()[0]
			})
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Bytes()[0] < tt.want[j].Bytes()[0]
			})
			assert.Equal(t, len(got), len(tt.want))
			for i, w := range tt.want {
				assert.DeepEqual(t, w.Bytes(), got[i].AggregationBits.Bytes())
			}
		}
		t.Run(fmt.Sprintf("%s/%s", tt.name, NaiveAggregation), func(t *testing.T) {
			resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
				AttestationAggregationStrategy: string(NaiveAggregation),
			})
			defer resetCfg()
			runner()
		})
		t.Run(fmt.Sprintf("%s/%s", tt.name, MaxCoverAggregation), func(t *testing.T) {
			resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
				AttestationAggregationStrategy: string(MaxCoverAggregation),
			})
			defer resetCfg()
			runner()
		})
	}
}
