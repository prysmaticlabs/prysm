package kv

import (
	"reflect"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
)

func TestKV_Unaggregated_AlreadyAggregated(t *testing.T) {
	cache := NewAttCaches()

	att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b111}}

	wanted := "attestation is aggregated"
	if err := cache.SaveUnaggregatedAttestation(att); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not received wanted error")
	}
}

func TestKV_Unaggregated_CanDelete(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b110}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveUnaggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	if err := cache.DeleteUnaggregatedAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := cache.DeleteUnaggregatedAttestation(att2); err != nil {
		t.Fatal(err)
	}

	if err := cache.DeleteUnaggregatedAttestation(att3); err != nil {
		t.Fatal(err)
	}

	returned := cache.UnaggregatedAttestations()

	if !reflect.DeepEqual([]*ethpb.Attestation{}, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}

func TestKV_Unaggregated_FilterBySlotsAndCommitteeIDs(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b110}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b110}}
	att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 2}, AggregationBits: bitfield.Bitlist{0b110}}

	atts := []*ethpb.Attestation{att1, att2, att3, att4}

	for _, att := range atts {
		if err := cache.SaveUnaggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.UnAggregatedAttestationsBySlotIndex(1, 1)

	if !reflect.DeepEqual([]*ethpb.Attestation{att1, att3}, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}

func TestKV_Unaggregated_BatchDelete(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b110}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b110}}
	att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 2}, AggregationBits: bitfield.Bitlist{0b110}}

	atts := []*ethpb.Attestation{att1, att2, att3, att4}

	for _, att := range atts {
		if err := cache.SaveUnaggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}
	if err := cache.DeleteUnaggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}

	returned := cache.UnaggregatedAttestations()
	if !reflect.DeepEqual([]*ethpb.Attestation{}, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}
