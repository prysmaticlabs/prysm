package kv

import (
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestKV_BlockAttestation_CanSaveRetrieve(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveBlockAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.BlockAttestations()

	sort.Slice(returned, func(i, j int) bool {
		return returned[i].Data.Slot < returned[j].Data.Slot
	})

	assert.DeepEqual(t, atts, returned)
}

func TestKV_BlockAttestation_CanDelete(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveBlockAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	if err := cache.DeleteBlockAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := cache.DeleteBlockAttestation(att3); err != nil {
		t.Fatal(err)
	}

	returned := cache.BlockAttestations()
	wanted := []*ethpb.Attestation{att2}
	assert.DeepEqual(t, wanted, returned)
}
