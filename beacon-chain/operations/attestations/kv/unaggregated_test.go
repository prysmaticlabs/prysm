package kv

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/params"
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

func TestKV_Unaggregated_CheckExpTime(t *testing.T) {
	cache := NewAttCaches()

	att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b11}}
	r, _ := ssz.HashTreeRoot(att)

	if err := cache.SaveUnaggregatedAttestation(att); err != nil {
		t.Fatal(err)
	}

	item, exp, exists := cache.unAggregatedAtt.GetWithExpiration(string(r[:]))
	if !exists {
		t.Error("Saved att does not exist")
	}

	receivedAtt := item.(*ethpb.Attestation)
	if !proto.Equal(att, receivedAtt) {
		t.Error("Did not receive correct unaggregated att")
	}

	wanted := float64(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	if math.RoundToEven(exp.Sub(time.Now()).Seconds()) != wanted {
		t.Errorf("Did not receive correct exp time. Wanted: %f, got: %f", wanted,
			math.RoundToEven(exp.Sub(time.Now()).Seconds()))
	}
}
