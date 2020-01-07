package attestations

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestBatchAttestations_Multiple(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	unaggregatedAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b101000}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b100010}, Signature: sig.Marshal()},
	}
	aggregatedAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b111000}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b100011}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b110001}, Signature: sig.Marshal()},
	}
	blockAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b100001}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 0}, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b111000}, Signature: sig.Marshal()}, // Duplicated
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b100011}, Signature: sig.Marshal()}, // Duplicated
	}
	if err := s.pool.SaveUnaggregatedAttestations(unaggregatedAtts); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.SaveAggregatedAttestations(aggregatedAtts); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.SaveBlockAttestations(blockAtts); err != nil {
		t.Fatal(err)
	}

	if err := s.batchForkChoiceAtts(context.Background()); err != nil {
		t.Fatal(err)
	}

	wanted, err := helpers.AggregateAttestations([]*ethpb.Attestation{unaggregatedAtts[0], aggregatedAtts[0], blockAtts[0]})
	if err != nil {
		t.Fatal(err)
	}
	aggregated, err := helpers.AggregateAttestations([]*ethpb.Attestation{unaggregatedAtts[1], aggregatedAtts[1], blockAtts[1]})
	if err != nil {
		t.Fatal(err)
	}
	wanted = append(wanted, aggregated...)
	aggregated, err = helpers.AggregateAttestations([]*ethpb.Attestation{unaggregatedAtts[2], aggregatedAtts[2], blockAtts[2]})
	if err != nil {
		t.Fatal(err)
	}

	wanted = append(wanted, aggregated...)
	received := s.pool.ForkchoiceAttestations()

	sort.Slice(received, func(i, j int) bool {
		return received[i].Data.Slot < received[j].Data.Slot
	})
	sort.Slice(wanted, func(i, j int) bool {
		return wanted[i].Data.Slot < wanted[j].Data.Slot
	})

	if !reflect.DeepEqual(wanted, received) {
		t.Error("Did not aggregation and save for batch")
	}
}

func TestBatchAttestations_Single(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	unaggregatedAtts := []*ethpb.Attestation{
		{AggregationBits: bitfield.Bitlist{0b101000}, Signature: sig.Marshal()},
		{AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
	}
	aggregatedAtts := []*ethpb.Attestation{
		{AggregationBits: bitfield.Bitlist{0b101100}, Signature: sig.Marshal()},
		{AggregationBits: bitfield.Bitlist{0b110010}, Signature: sig.Marshal()},
	}
	blockAtts := []*ethpb.Attestation{
		{AggregationBits: bitfield.Bitlist{0b110010}, Signature: sig.Marshal()},
		{AggregationBits: bitfield.Bitlist{0b100010}, Signature: sig.Marshal()},
		{AggregationBits: bitfield.Bitlist{0b110010}, Signature: sig.Marshal()}, // Duplicated
	}
	if err := s.pool.SaveUnaggregatedAttestations(unaggregatedAtts); err != nil {
		t.Fatal(err)
	}

	if err := s.pool.SaveAggregatedAttestations(aggregatedAtts); err != nil {
		t.Fatal(err)
	}

	if err := s.pool.SaveBlockAttestations(blockAtts); err != nil {
		t.Fatal(err)
	}

	if err := s.batchForkChoiceAtts(context.Background()); err != nil {
		t.Fatal(err)
	}

	wanted, err := helpers.AggregateAttestations(append(unaggregatedAtts, aggregatedAtts...))
	if err != nil {
		t.Fatal(err)
	}

	wanted, err = helpers.AggregateAttestations(append(wanted, blockAtts...))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(wanted, s.pool.ForkchoiceAttestations()) {
		t.Error("Did not aggregation and save for batch")
	}
}

func TestAggregateAndSaveForkChoiceAtts_Single(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	atts := []*ethpb.Attestation{
		{AggregationBits: bitfield.Bitlist{0b101}, Signature: sig.Marshal()},
		{AggregationBits: bitfield.Bitlist{0b110}, Signature: sig.Marshal()}}
	if err := s.aggregateAndSaveForkChoiceAtts(atts); err != nil {
		t.Fatal(err)
	}

	wanted, err := helpers.AggregateAttestations(atts)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(wanted, s.pool.ForkchoiceAttestations()) {
		t.Error("Did not aggregation and save")
	}
}

func TestAggregateAndSaveForkChoiceAtts_Multiple(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	atts1 := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0b101}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0b110}, Signature: sig.Marshal()},
	}
	if err := s.aggregateAndSaveForkChoiceAtts(atts1); err != nil {
		t.Fatal(err)
	}
	atts2 := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10110}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11000}, Signature: sig.Marshal()},
	}
	if err := s.aggregateAndSaveForkChoiceAtts(atts2); err != nil {
		t.Fatal(err)
	}
	att3 := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1100}, Signature: sig.Marshal()},
	}
	if err := s.aggregateAndSaveForkChoiceAtts(att3); err != nil {
		t.Fatal(err)
	}

	wanted, err := helpers.AggregateAttestations(atts1)
	if err != nil {
		t.Fatal(err)
	}
	aggregated, err := helpers.AggregateAttestations(atts2)
	if err != nil {
		t.Fatal(err)
	}

	wanted = append(wanted, aggregated...)
	wanted = append(wanted, att3...)

	received := s.pool.ForkchoiceAttestations()
	sort.Slice(received, func(i, j int) bool {
		return received[i].Data.Slot < received[j].Data.Slot
	})
	if !reflect.DeepEqual(wanted, received) {
		t.Error("Did not aggregation and save")
	}
}

func TestSeenAttestations_PresentInCache(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	att1 := &ethpb.Attestation{Signature: []byte{'A'}}
	got, err := s.seen(att1)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Wanted false, got true")
	}

	time.Sleep(100 * time.Millisecond)
	got, err = s.seen(att1)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Wanted true, got false")
	}
}
