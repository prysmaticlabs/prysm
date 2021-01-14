package validator

import (
	"bytes"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProposer_ProposerAtts_sortByProfitability(t *testing.T) {
	atts := proposerAtts([]*ethpb.Attestation{
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 4}, AggregationBits: bitfield.Bitlist{0b11100000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11000000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b11100000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 4}, AggregationBits: bitfield.Bitlist{0b11110000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11100000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b11000000}}),
	})
	want := proposerAtts([]*ethpb.Attestation{
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 4}, AggregationBits: bitfield.Bitlist{0b11110000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 4}, AggregationBits: bitfield.Bitlist{0b11100000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b11000000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b11100000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11100000}}),
		testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11000000}}),
	})
	atts = atts.sortByProfitability()
	require.DeepEqual(t, want, atts)
}

func TestProposer_ProposerAtts_dedup(t *testing.T) {
	data1 := testutil.HydrateAttestationData(&ethpb.AttestationData{
		Slot: 4,
	})
	data2 := testutil.HydrateAttestationData(&ethpb.AttestationData{
		Slot: 5,
	})
	tests := []struct {
		name string
		atts proposerAtts
		want proposerAtts
	}{
		{
			name: "nil list",
			atts: nil,
			want: proposerAtts(nil),
		},
		{
			name: "empty list",
			atts: proposerAtts{},
			want: proposerAtts{},
		},
		{
			name: "single item",
			atts: proposerAtts{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{}},
			},
			want: proposerAtts{
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{}},
			},
		},
		{
			name: "two items no duplicates",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10111110, 0x01}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01111111, 0x01}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01111111, 0x01}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10111110, 0x01}},
			},
		},
		{
			name: "two items with duplicates",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0xba, 0x01}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0xba, 0x01}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0xba, 0x01}},
			},
		},
		{
			name: "sorted no duplicates",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00101011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100000, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00101011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100000, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
			},
		},
		{
			name: "sorted with duplicates",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
		},
		{
			name: "all equal",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
			},
		},
		{
			name: "unsorted no duplicates",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00100010, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00100010, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00010000, 0b1}},
			},
		},
		{
			name: "unsorted with duplicates",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10100101, 0b1}},
			},
		},
		{
			name: "no proper subset (same root)",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10000001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00011001, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00011001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b10000001, 0b1}},
			},
		},
		{
			name: "proper subset (same root)",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
		},
		{
			name: "no proper subset (different root)",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000101, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b10000001, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00011001, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00011001, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b10000001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000101, 0b1}},
			},
		},
		{
			name: "proper subset (different root 1)",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00000011, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00000001, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b01101101, 0b1}},
			},
		},
		{
			name: "proper subset (different root 2)",
			atts: proposerAtts{
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b00001111, 0b1}},
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
			},
			want: proposerAtts{
				&ethpb.Attestation{Data: data2, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
				&ethpb.Attestation{Data: data1, AggregationBits: bitfield.Bitlist{0b11001111, 0b1}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atts := tt.atts.dedup()
			sort.Slice(atts, func(i, j int) bool {
				if atts[i].AggregationBits.Count() == atts[j].AggregationBits.Count() {
					if atts[i].Data.Slot == atts[j].Data.Slot {
						return bytes.Compare(atts[i].AggregationBits, atts[j].AggregationBits) <= 0
					}
					return atts[i].Data.Slot > atts[j].Data.Slot
				}
				return atts[i].AggregationBits.Count() > atts[j].AggregationBits.Count()
			})
			assert.DeepEqual(t, tt.want, atts)
		})
	}
}
