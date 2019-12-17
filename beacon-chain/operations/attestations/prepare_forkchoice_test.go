package attestations

import (
	"context"
	"reflect"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func Test_AggregateAndSaveForkChoiceAtts_Single(t *testing.T) {
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

func Test_AggregateAndSaveForkChoiceAtts_Multiple(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	atts1 := []*ethpb.Attestation{
		{AggregationBits: bitfield.Bitlist{0b101}, Signature: sig.Marshal()},
		{AggregationBits: bitfield.Bitlist{0b110}, Signature: sig.Marshal()},
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

	wanted1, err := helpers.AggregateAttestations(atts1)
	if err != nil {
		t.Fatal(err)
	}
	wanted2, err := helpers.AggregateAttestations(atts2)
	if err != nil {
		t.Fatal(err)
	}

	wanted1 = append(wanted1, wanted2...)
	wanted1 = append(wanted1, att3...)
	if !reflect.DeepEqual(wanted1, s.pool.ForkchoiceAttestations()) {
		t.Error("Did not aggregation and save")
	}
}

func Test_SeenAttestations_PresentInCache(t *testing.T) {
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
