package attestations

import (
	"context"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPruneExpired_Ticker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s, err := NewService(ctx, &Config{
		Pool:          NewPool(),
		pruneInterval: 250 * time.Millisecond,
	})
	require.NoError(t, err)

	atts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b1000, 0b1}},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1000, 0b1}},
	}
	if err := s.pool.SaveUnaggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}
	if s.pool.UnaggregatedAttestationCount() != 2 {
		t.Fatalf("Unexpected number of attestations: %d", s.pool.UnaggregatedAttestationCount())
	}
	atts = []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b1101, 0b1}},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101, 0b1}},
	}
	if err := s.pool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, s.pool.AggregatedAttestationCount())
	if err := s.pool.SaveBlockAttestations(atts); err != nil {
		t.Fatal(err)
	}

	// Rewind back one epoch worth of time.
	s.genesisTime = uint64(roughtime.Now().Unix()) - params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot

	go s.pruneAttsPool()

	done := make(chan struct{}, 1)
	runutil.RunEvery(ctx, 500*time.Millisecond, func() {
		for _, attestation := range s.pool.UnaggregatedAttestations() {
			if attestation.Data.Slot == 0 {
				return
			}
		}
		for _, attestation := range s.pool.AggregatedAttestations() {
			if attestation.Data.Slot == 0 {
				return
			}
		}
		for _, attestation := range s.pool.BlockAttestations() {
			if attestation.Data.Slot == 0 {
				return
			}
		}
		if s.pool.UnaggregatedAttestationCount() != 1 || s.pool.AggregatedAttestationCount() != 1 {
			return
		}
		done <- struct{}{}
	})
	select {
	case <-done:
		// All checks are passed.
	case <-ctx.Done():
		t.Error("Test case takes too long to complete")
	}
}

func TestPruneExpired_PruneExpiredAtts(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

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

func TestPruneExpired_Expired(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

	// Rewind back one epoch worth of time.
	s.genesisTime = uint64(roughtime.Now().Unix()) - params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot
	if !s.expired(0) {
		t.Error("Should expired")
	}
	if s.expired(1) {
		t.Error("Should not expired")
	}
}
