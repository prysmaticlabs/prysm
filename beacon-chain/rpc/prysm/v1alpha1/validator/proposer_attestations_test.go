package validator

import (
	"bytes"
	"context"
	"sort"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls/blst"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func TestProposer_ProposerAtts_sort(t *testing.T) {
	feat := features.Get()
	feat.DisableCommitteeAwarePacking = true
	reset := features.InitWithReset(feat)
	defer reset()

	type testData struct {
		slot primitives.Slot
		bits bitfield.Bitlist
	}
	getAtts := func(data []testData) proposerAtts {
		var atts proposerAtts
		for _, att := range data {
			atts = append(atts, util.HydrateAttestation(&ethpb.Attestation{
				Data: &ethpb.AttestationData{Slot: att.slot}, AggregationBits: att.bits}))
		}
		return atts
	}

	t.Run("no atts", func(t *testing.T) {
		atts := getAtts([]testData{})
		want := getAtts([]testData{})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("single att", func(t *testing.T) {
		atts := getAtts([]testData{
			{4, bitfield.Bitlist{0b11100000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11100000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("single att per slot", func(t *testing.T) {
		atts := getAtts([]testData{
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("two atts on one of the slots", func(t *testing.T) {
		atts := getAtts([]testData{
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{4, bitfield.Bitlist{0b11110000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11110000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("compare to native sort", func(t *testing.T) {
		// The max-cover based approach will select 0b00001100 instead, despite lower bit count
		// (since it has two new/unknown bits).
		t.Run("max-cover", func(t *testing.T) {
			atts := getAtts([]testData{
				{1, bitfield.Bitlist{0b11000011, 0b1}},
				{1, bitfield.Bitlist{0b11001000, 0b1}},
				{1, bitfield.Bitlist{0b00001100, 0b1}},
			})
			want := getAtts([]testData{
				{1, bitfield.Bitlist{0b11000011, 0b1}},
				{1, bitfield.Bitlist{0b00001100, 0b1}},
				{1, bitfield.Bitlist{0b11001000, 0b1}},
			})
			atts, err := atts.sort()
			if err != nil {
				t.Error(err)
			}
			require.DeepEqual(t, want, atts)
		})
	})

	t.Run("multiple slots", func(t *testing.T) {
		atts := getAtts([]testData{
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{4, bitfield.Bitlist{0b11110000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11110000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("follows max-cover", func(t *testing.T) {
		// Items at slot 4, must be first split into two lists by max-cover, with
		// 0b10000011 scoring higher (as it provides more info in addition to already selected
		// attestations) than 0b11100001 (despite naive bit count suggesting otherwise). Then,
		// both selected and non-selected attestations must be additionally sorted by bit count.
		atts := getAtts([]testData{
			{4, bitfield.Bitlist{0b00000001, 0b1}},
			{4, bitfield.Bitlist{0b11100001, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{4, bitfield.Bitlist{0b10000011, 0b1}},
			{4, bitfield.Bitlist{0b11111000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11111000, 0b1}},
			{4, bitfield.Bitlist{0b10000011, 0b1}},
			{4, bitfield.Bitlist{0b11100001, 0b1}},
			{4, bitfield.Bitlist{0b00000001, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})
}

func TestProposer_ProposerAtts_committeeAwareSort(t *testing.T) {
	type testData struct {
		slot primitives.Slot
		bits bitfield.Bitlist
	}
	getAtts := func(data []testData) proposerAtts {
		var atts proposerAtts
		for _, att := range data {
			atts = append(atts, util.HydrateAttestation(&ethpb.Attestation{
				Data: &ethpb.AttestationData{Slot: att.slot}, AggregationBits: att.bits}))
		}
		return atts
	}

	t.Run("no atts", func(t *testing.T) {
		atts := getAtts([]testData{})
		want := getAtts([]testData{})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("single att", func(t *testing.T) {
		atts := getAtts([]testData{
			{4, bitfield.Bitlist{0b11100000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11100000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("single att per slot", func(t *testing.T) {
		atts := getAtts([]testData{
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("two atts on one of the slots", func(t *testing.T) {

		atts := getAtts([]testData{
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{4, bitfield.Bitlist{0b11110000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11110000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("compare to native sort", func(t *testing.T) {
		// The max-cover based approach will select 0b00001100 instead, despite lower bit count
		// (since it has two new/unknown bits).
		t.Run("max-cover", func(t *testing.T) {
			atts := getAtts([]testData{
				{1, bitfield.Bitlist{0b11000011, 0b1}},
				{1, bitfield.Bitlist{0b11001000, 0b1}},
				{1, bitfield.Bitlist{0b00001100, 0b1}},
			})
			want := getAtts([]testData{
				{1, bitfield.Bitlist{0b11000011, 0b1}},
				{1, bitfield.Bitlist{0b00001100, 0b1}},
			})
			atts, err := atts.sort()
			if err != nil {
				t.Error(err)
			}
			require.DeepEqual(t, want, atts)
		})
	})

	t.Run("multiple slots", func(t *testing.T) {
		atts := getAtts([]testData{
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{4, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{4, bitfield.Bitlist{0b11110000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11110000, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})

	t.Run("follows max-cover", func(t *testing.T) {
		// Items at slot 4 must be first split into two lists by max-cover, with
		// 0b10000011 being selected and 0b11100001 being leftover (despite naive bit count suggesting otherwise).
		atts := getAtts([]testData{
			{4, bitfield.Bitlist{0b00000001, 0b1}},
			{4, bitfield.Bitlist{0b11100001, 0b1}},
			{1, bitfield.Bitlist{0b11000000, 0b1}},
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{4, bitfield.Bitlist{0b10000011, 0b1}},
			{4, bitfield.Bitlist{0b11111000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
		})
		want := getAtts([]testData{
			{4, bitfield.Bitlist{0b11111000, 0b1}},
			{4, bitfield.Bitlist{0b10000011, 0b1}},
			{3, bitfield.Bitlist{0b11000000, 0b1}},
			{2, bitfield.Bitlist{0b11100000, 0b1}},
			{1, bitfield.Bitlist{0b11100000, 0b1}},
		})
		atts, err := atts.sort()
		if err != nil {
			t.Error(err)
		}
		require.DeepEqual(t, want, atts)
	})
}

func TestProposer_sort_DifferentCommittees(t *testing.T) {
	t.Run("one att per committee", func(t *testing.T) {
		c1_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11111000, 0b1}, Data: &ethpb.AttestationData{CommitteeIndex: 1}})
		c2_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11100000, 0b1}, Data: &ethpb.AttestationData{CommitteeIndex: 2}})
		atts := proposerAtts{c1_a1, c2_a1}
		atts, err := atts.sort()
		require.NoError(t, err)
		want := proposerAtts{c1_a1, c2_a1}
		assert.DeepEqual(t, want, atts)
	})
	t.Run("multiple atts per committee", func(t *testing.T) {
		c1_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11111100, 0b1}, Data: &ethpb.AttestationData{CommitteeIndex: 1}})
		c1_a2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10000010, 0b1}, Data: &ethpb.AttestationData{CommitteeIndex: 1}})
		c2_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11110000, 0b1}, Data: &ethpb.AttestationData{CommitteeIndex: 2}})
		c2_a2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11100000, 0b1}, Data: &ethpb.AttestationData{CommitteeIndex: 2}})
		atts := proposerAtts{c1_a1, c1_a2, c2_a1, c2_a2}
		atts, err := atts.sort()
		require.NoError(t, err)

		want := proposerAtts{c1_a1, c2_a1, c1_a2}
		assert.DeepEqual(t, want, atts)
	})
	t.Run("multiple atts per committee, multiple slots", func(t *testing.T) {
		s2_c1_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11111100, 0b1}, Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 1}})
		s2_c1_a2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10000010, 0b1}, Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 1}})
		s2_c2_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11110000, 0b1}, Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 2}})
		s2_c2_a2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11000000, 0b1}, Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 2}})
		s1_c1_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11111100, 0b1}, Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}})
		s1_c1_a2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10000010, 0b1}, Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}})
		s1_c2_a1 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11110000, 0b1}, Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 2}})
		s1_c2_a2 := util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11000000, 0b1}, Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 2}})

		// Arrange in some random order
		atts := proposerAtts{s1_c1_a1, s2_c1_a2, s1_c2_a2, s2_c2_a2, s1_c2_a1, s2_c2_a1, s1_c1_a2, s2_c1_a1}

		atts, err := atts.sort()
		require.NoError(t, err)

		want := proposerAtts{s2_c1_a1, s2_c2_a1, s2_c1_a2, s1_c1_a1, s1_c2_a1, s1_c1_a2}
		assert.DeepEqual(t, want, atts)
	})
}

func TestProposer_ProposerAtts_dedup(t *testing.T) {
	data1 := util.HydrateAttestationData(&ethpb.AttestationData{
		Slot: 4,
	})
	data2 := util.HydrateAttestationData(&ethpb.AttestationData{
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
			atts, err := tt.atts.dedup()
			if err != nil {
				t.Error(err)
			}
			sort.Slice(atts, func(i, j int) bool {
				if atts[i].GetAggregationBits().Count() == atts[j].GetAggregationBits().Count() {
					if atts[i].GetData().Slot == atts[j].GetData().Slot {
						return bytes.Compare(atts[i].GetAggregationBits(), atts[j].GetAggregationBits()) <= 0
					}
					return atts[i].GetData().Slot > atts[j].GetData().Slot
				}
				return atts[i].GetAggregationBits().Count() > atts[j].GetAggregationBits().Count()
			})
			assert.DeepEqual(t, tt.want, atts)
		})
	}
}

func Test_packAttestations(t *testing.T) {
	ctx := context.Background()
	phase0Att := &ethpb.Attestation{
		AggregationBits: bitfield.Bitlist{0b11111},
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: make([]byte, 32),
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
		},
		Signature: make([]byte, 96),
	}
	cb := primitives.NewAttestationCommitteeBits()
	cb.SetBitAt(0, true)
	key, err := blst.RandKey()
	require.NoError(t, err)
	sig := key.Sign([]byte{'X'})
	electraAtt := &ethpb.AttestationElectra{
		AggregationBits: bitfield.Bitlist{0b11111},
		CommitteeBits:   cb,
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: make([]byte, 32),
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
		},
		Signature: sig.Marshal(),
	}
	pool := attestations.NewPool()
	require.NoError(t, pool.SaveAggregatedAttestations([]ethpb.Att{phase0Att, electraAtt}))
	slot := primitives.Slot(1)
	s := &Server{AttPool: pool, HeadFetcher: &chainMock.ChainService{}, TimeFetcher: &chainMock.ChainService{Slot: &slot}}

	t.Run("Phase 0", func(t *testing.T) {
		st, _ := util.DeterministicGenesisState(t, 64)
		require.NoError(t, st.SetSlot(1))

		atts, err := s.packAttestations(ctx, st, 0)
		require.NoError(t, err)
		require.Equal(t, 1, len(atts))
		assert.DeepEqual(t, phase0Att, atts[0])
	})
	t.Run("Electra", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig().Copy()
		cfg.ElectraForkEpoch = 1
		params.OverrideBeaconConfig(cfg)

		st, _ := util.DeterministicGenesisStateElectra(t, 64)
		require.NoError(t, st.SetSlot(params.BeaconConfig().SlotsPerEpoch+1))

		atts, err := s.packAttestations(ctx, st, params.BeaconConfig().SlotsPerEpoch)
		require.NoError(t, err)
		require.Equal(t, 1, len(atts))
		assert.DeepEqual(t, electraAtt, atts[0])
	})
	t.Run("Electra block with Deneb state", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig().Copy()
		cfg.ElectraForkEpoch = 1
		params.OverrideBeaconConfig(cfg)

		st, _ := util.DeterministicGenesisStateDeneb(t, 64)
		require.NoError(t, st.SetSlot(params.BeaconConfig().SlotsPerEpoch+1))

		atts, err := s.packAttestations(ctx, st, params.BeaconConfig().SlotsPerEpoch)
		require.NoError(t, err)
		require.Equal(t, 1, len(atts))
		assert.DeepEqual(t, electraAtt, atts[0])
	})
}

func Test_limitToMaxAttestations(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		atts := make([]ethpb.Att, params.BeaconConfig().MaxAttestations+1)
		for i := range atts {
			atts[i] = &ethpb.Attestation{}
		}

		// 1 less than limit
		pAtts := proposerAtts(atts[:len(atts)-3])
		assert.Equal(t, len(pAtts), len(pAtts.limitToMaxAttestations()))

		// equal to limit
		pAtts = atts[:len(atts)-2]
		assert.Equal(t, len(pAtts), len(pAtts.limitToMaxAttestations()))

		// 1 more than limit
		pAtts = atts
		assert.Equal(t, len(pAtts)-1, len(pAtts.limitToMaxAttestations()))
	})
	t.Run("Electra", func(t *testing.T) {
		atts := make([]ethpb.Att, params.BeaconConfig().MaxAttestationsElectra+1)
		for i := range atts {
			atts[i] = &ethpb.AttestationElectra{}
		}

		// 1 less than limit
		pAtts := proposerAtts(atts[:len(atts)-3])
		assert.Equal(t, len(pAtts), len(pAtts.limitToMaxAttestations()))

		// equal to limit
		pAtts = atts[:len(atts)-2]
		assert.Equal(t, len(pAtts), len(pAtts.limitToMaxAttestations()))

		// 1 more than limit
		pAtts = atts
		assert.Equal(t, len(pAtts)-1, len(pAtts.limitToMaxAttestations()))
	})
}

func Test_filterBatchSignature(t *testing.T) {
	st, k := util.DeterministicGenesisState(t, 64)
	// Generate 1 good signature
	aGood, err := util.GenerateAttestations(st, k, 1, 0, false)
	require.NoError(t, err)
	// Generate 1 bad signature
	aBad := util.NewAttestation()
	pa := proposerAtts(aGood)
	pa = append(pa, aBad)
	aFiltered := pa.filterBatchSignature(context.Background(), st)
	assert.Equal(t, 1, len(aFiltered))
	assert.DeepEqual(t, aGood[0], aFiltered[0])
}

func Test_isAttestationFromCurrentEpoch(t *testing.T) {
	slot := primitives.Slot(1)
	epoch := slots.ToEpoch(slot)
	s := &Server{}
	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}},
	}
	require.Equal(t, true, s.isAttestationFromCurrentEpoch(a, epoch))

	a.Data.Target.Epoch = 1
	require.Equal(t, false, s.isAttestationFromCurrentEpoch(a, epoch))
}

func Test_isAttestationFromPreviousEpoch(t *testing.T) {
	slot := params.BeaconConfig().SlotsPerEpoch
	epoch := slots.ToEpoch(slot)
	s := &Server{}
	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}},
	}
	require.Equal(t, true, s.isAttestationFromPreviousEpoch(a, epoch))

	a.Data.Target.Epoch = 1
	require.Equal(t, false, s.isAttestationFromPreviousEpoch(a, epoch))
}

func Test_filterCurrentEpochAttestationByTarget(t *testing.T) {
	slot := params.BeaconConfig().SlotsPerEpoch
	epoch := slots.ToEpoch(slot)
	s := &Server{}
	targetRoot := [32]byte{'a'}
	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot: 1,
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  targetRoot[:],
			},
		},
	}
	got, err := s.filterCurrentEpochAttestationByTarget(a, targetRoot, 1, epoch)
	require.NoError(t, err)
	require.Equal(t, true, got)

	got, err = s.filterCurrentEpochAttestationByTarget(a, [32]byte{}, 1, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)

	got, err = s.filterCurrentEpochAttestationByTarget(a, targetRoot, 2, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)

	a.Data.Target.Epoch = 2
	got, err = s.filterCurrentEpochAttestationByTarget(a, targetRoot, 1, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)
}

func Test_filterPreviousEpochAttestationByTarget(t *testing.T) {
	slot := 2 * params.BeaconConfig().SlotsPerEpoch
	epoch := slots.ToEpoch(slot)
	s := &Server{}
	targetRoot := [32]byte{'a'}
	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot: 1,
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  targetRoot[:],
			},
		},
	}
	got, err := s.filterPreviousEpochAttestationByTarget(a, &ethpb.Checkpoint{
		Epoch: 1,
		Root:  targetRoot[:],
	}, epoch)
	require.NoError(t, err)
	require.Equal(t, true, got)

	got, err = s.filterPreviousEpochAttestationByTarget(a, &ethpb.Checkpoint{
		Epoch: 1,
	}, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)

	got, err = s.filterPreviousEpochAttestationByTarget(a, &ethpb.Checkpoint{
		Epoch: 2,
		Root:  targetRoot[:],
	}, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)

	got, err = s.filterPreviousEpochAttestationByTarget(a, &ethpb.Checkpoint{
		Epoch: 3,
		Root:  targetRoot[:],
	}, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)
}

func Test_filterCurrentEpochAttestationByForkchoice(t *testing.T) {
	slot := params.BeaconConfig().SlotsPerEpoch
	epoch := slots.ToEpoch(slot)
	s := &Server{}
	targetRoot := [32]byte{'a'}
	a := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: make([]byte, 32),
			Slot:            params.BeaconConfig().SlotsPerEpoch,
			Target: &ethpb.Checkpoint{
				Epoch: 1,
				Root:  targetRoot[:],
			},
		},
	}

	ctx := context.Background()
	got, err := s.filterCurrentEpochAttestationByForkchoice(ctx, a, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)

	a.Data.BeaconBlockRoot = targetRoot[:]
	s.ForkchoiceFetcher = &chainMock.ChainService{BlockSlot: 1}
	got, err = s.filterCurrentEpochAttestationByForkchoice(ctx, a, epoch)
	require.NoError(t, err)
	require.Equal(t, true, got)

	s.ForkchoiceFetcher = &chainMock.ChainService{BlockSlot: 100}
	got, err = s.filterCurrentEpochAttestationByForkchoice(ctx, a, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)

	slot = params.BeaconConfig().SlotsPerEpoch * 2
	epoch = slots.ToEpoch(slot)
	got, err = s.filterCurrentEpochAttestationByForkchoice(ctx, a, epoch)
	require.NoError(t, err)
	require.Equal(t, false, got)
}
