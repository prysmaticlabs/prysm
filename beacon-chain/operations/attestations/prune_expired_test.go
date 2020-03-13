package attestations

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

func TestPruneExpiredAtts_CanPrune(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b1111}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1110}}
	atts := []*ethpb.Attestation{att1, att2, att3, att4}
	if err := s.pool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.SaveBlockAttestations(atts); err != nil {
		t.Fatal(err)
	}

	// Rewind back one epoch worth of time.
	s.genesisTime = uint64(roughtime.Now().Unix()) - params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot

	s.pruneExpiredAtts()
	// All the attestations on slot 0 should be pruned.
	for _, attestation := range s.pool.AggregatedAttestations() {
		if attestation.Data.Slot == 0 {
			t.Error("Should be pruned")
		}
	}
	for _, attestation := range s.pool.BlockAttestations() {
		if attestation.Data.Slot == 0 {
			t.Error("Should be pruned")
		}
	}
}

func TestExpired_AttsCanExpire(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	// Rewind back one epoch worth of time.
	s.genesisTime = uint64(roughtime.Now().Unix()) - params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot
	if !s.expired(0) {
		t.Error("Should expired")
	}
	if s.expired(1) {
		t.Error("Should not expired")
	}
}
