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

func TestKV_Forkchoice_CanSaveRetrieve(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveForkchoiceAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.ForkchoiceAttestations()

	sort.Slice(returned, func(i, j int) bool {
		return returned[i].Data.Slot < returned[j].Data.Slot
	})

	if !reflect.DeepEqual(atts, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}

func TestKV_Forkchoice_CanDelete(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveForkchoiceAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	if err := cache.DeleteForkchoiceAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := cache.DeleteForkchoiceAttestation(att3); err != nil {
		t.Fatal(err)
	}

	returned := cache.ForkchoiceAttestations()
	wanted := []*ethpb.Attestation{att2}

	if !reflect.DeepEqual(wanted, returned) {
		t.Error("Did not receive correct aggregated atts")
	}
}

func TestKV_Forkchoice_CheckExpTime(t *testing.T) {
	cache := NewAttCaches()

	att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b111}}
	r, _ := ssz.HashTreeRoot(att)

	if err := cache.SaveForkchoiceAttestation(att); err != nil {
		t.Fatal(err)
	}

	item, exp, exists := cache.forkchoiceAtt.GetWithExpiration(string(r[:]))
	if !exists {
		t.Error("Saved att does not exist")
	}

	receivedAtt := item.(*ethpb.Attestation)
	if !proto.Equal(att, receivedAtt) {
		t.Error("Did not receive correct aggregated att")
	}

	wanted := float64(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	if math.RoundToEven(exp.Sub(time.Now()).Seconds()) != wanted {
		t.Errorf("Did not receive correct exp time. Wanted: %f, got: %f", wanted,
			math.RoundToEven(exp.Sub(time.Now()).Seconds()))
	}
}
