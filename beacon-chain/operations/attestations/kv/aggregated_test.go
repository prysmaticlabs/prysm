package kv

import (
	"math"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestKV_Aggregated_NotAggregated(t *testing.T) {
	cache := NewAttCaches()

	att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11}}

	if err := cache.SaveAggregatedAttestation(att); err != nil {
		t.Error(err)
	}
	if ok, _ := cache.HasAggregatedAttestation(att); !ok {
		t.Error("Expected cache to have attestation")
	}
}

func TestKV_Aggregated_CanSaveRetrieve(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveAggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.AggregatedAttestations()

	sort.Slice(returned, func(i, j int) bool {
		return returned[i].Data.Slot < returned[j].Data.Slot
	})

	if !reflect.DeepEqual(atts, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}

func TestKV_Aggregated_CanDelete(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveAggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	if err := cache.DeleteAggregatedAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := cache.DeleteAggregatedAttestation(att3); err != nil {
		t.Fatal(err)
	}

	returned := cache.AggregatedAttestations()
	wanted := []*ethpb.Attestation{att2}

	if !reflect.DeepEqual(wanted, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}

func TestKV_Aggregated_CheckExpTime(t *testing.T) {
	cache := NewAttCaches()

	att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b111}}
	r, _ := ssz.HashTreeRoot(att.Data)

	if err := cache.SaveAggregatedAttestation(att); err != nil {
		t.Fatal(err)
	}

	item, exp, exists := cache.aggregatedAtt.GetWithExpiration(string(r[:]))
	if !exists {
		t.Error("Saved att does not exist")
	}

	receivedAtt := item.([]*ethpb.Attestation)[0]
	if !proto.Equal(att, receivedAtt) {
		t.Error("Did not receive correct aggregated att")
	}

	wanted := float64(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	if math.RoundToEven(exp.Sub(time.Now()).Seconds()) != wanted {
		t.Errorf("Did not receive correct exp time. Wanted: %f, got: %f", wanted,
			math.RoundToEven(exp.Sub(time.Now()).Seconds()))
	}
}

func TestKV_HasAggregatedAttestation(t *testing.T) {
	tests := []struct {
		name     string
		existing []*ethpb.Attestation
		input    *ethpb.Attestation
		want     bool
	}{
		{
			name: "empty cache aggregated",
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: false,
		},
		{
			name: "empty cache unaggregated",
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1001}},
			want: false,
		},
		{
			name: "single attestation in cache with exact match",
			existing: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: true,
		},
		{
			name: "single attestation in cache with subset aggregation",
			existing: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1110}},
			want: true,
		},
		{
			name: "single attestation in cache with superset aggregation",
			existing: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1110}},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: false,
		},
		{
			name: "multiple attestations with same data in cache with overlapping aggregation, input is subset",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1100111},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1100000}},
			want: true,
		},
		{
			name: "multiple attestations with same data in cache with overlapping aggregation and input is superset",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1100111},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111111}},
			want: false,
		},
		{
			name: "multiple attestations with different data in cache",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 2,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
				{
					Data: &ethpb.AttestationData{
						Slot: 3,
					},
					AggregationBits: bitfield.Bitlist{0b1100111},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111111}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			if err := cache.SaveAggregatedAttestations(tt.existing); err != nil {
				t.Error(err)
			}

			result, err := cache.HasAggregatedAttestation(tt.input)
			if err != nil {
				t.Error(err)
			}
			if result != tt.want {
				t.Errorf("Result = %v, wanted = %v", result, tt.want)
			}

			// Same test for block attestations
			cache = NewAttCaches()
			if err := cache.SaveBlockAttestations(tt.existing); err != nil {
				t.Error(err)
			}

			result, err = cache.HasAggregatedAttestation(tt.input)
			if err != nil {
				t.Error(err)
			}
			if result != tt.want {
				t.Errorf("Result = %v, wanted = %v", result, tt.want)
			}
		})
	}
}
