package kv

import (
	"reflect"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestKV_Aggregated_AggregateUnaggregatedAttestations(t *testing.T) {
	cache := NewAttCaches()
	priv := bls.RandKey()
	sig1 := priv.Sign([]byte{'a'})
	sig2 := priv.Sign([]byte{'b'})
	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig1.Marshal()}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1010}, Signature: sig1.Marshal()}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1100}, Signature: sig1.Marshal()}
	att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig2.Marshal()}
	att5 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig1.Marshal()}
	att6 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1010}, Signature: sig1.Marshal()}
	att7 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1100}, Signature: sig1.Marshal()}
	att8 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig2.Marshal()}
	atts := []*ethpb.Attestation{att1, att2, att3, att4, att5, att6, att7, att8}
	if err := cache.SaveUnaggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}
	if err := cache.AggregateUnaggregatedAttestations(); err != nil {
		t.Fatal(err)
	}

	if len(cache.AggregatedAttestationsBySlotIndex(1, 0)) != 1 {
		t.Fatal("Did not aggregate correctly")
	}
	if len(cache.AggregatedAttestationsBySlotIndex(2, 0)) != 1 {
		t.Fatal("Did not aggregate correctly")
	}
}

func TestKV_Aggregated_AggregatedAttestations(t *testing.T) {
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

func TestKV_Aggregated_DeleteAggregatedAttestation(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
	att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b10101}}
	atts := []*ethpb.Attestation{att1, att2, att3, att4}

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

func TestKV_Aggregated_HasAggregatedAttestation(t *testing.T) {
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

// todo remove
func TestKV_Aggregated_Aggregated_AggregatesAttestations(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1111}}
	atts := []*ethpb.Attestation{att1, att2}

	for _, att := range atts {
		if err := cache.SaveAggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.AggregatedAttestations()

	// It should have only returned att2.
	if !reflect.DeepEqual(att2, returned[0]) || len(returned) != 1 {
		t.Error("Did not receive correct aggregated atts")
	}
}
