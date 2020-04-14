package attestations

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestAggregateAttestations_SingleAttestation(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	unaggregatedAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0b100001}, Signature: sig.Marshal()},
	}

	if err := s.aggregateAttestations(context.Background(), unaggregatedAtts); err != nil {
		t.Fatal(err)
	}

	if len(s.pool.AggregatedAttestations()) != 0 {
		t.Error("Nothing should be aggregated")
	}

	if len(s.pool.UnaggregatedAttestations()) != 0 {
		t.Error("Unaggregated pool should be empty")
	}
}

func TestAggregateAttestations_MultipleAttestationsSameRoot(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{},
		Target: &ethpb.Checkpoint{},
	}
	attsToBeAggregated := []*ethpb.Attestation{
		{Data: data, AggregationBits: bitfield.Bitlist{0b110001}, Signature: sig.Marshal()},
		{Data: data, AggregationBits: bitfield.Bitlist{0b100010}, Signature: sig.Marshal()},
		{Data: data, AggregationBits: bitfield.Bitlist{0b101100}, Signature: sig.Marshal()},
	}

	if err := s.aggregateAttestations(context.Background(), attsToBeAggregated); err != nil {
		t.Fatal(err)
	}

	if len(s.pool.UnaggregatedAttestations()) != 0 {
		t.Error("Nothing should be unaggregated")
	}

	wanted, err := helpers.AggregateAttestations(attsToBeAggregated)
	if err != nil {
		t.Fatal(err)
	}
	got := s.pool.AggregatedAttestations()
	if !reflect.DeepEqual(wanted, got) {
		diff, _ := messagediff.PrettyDiff(got[0], wanted[0])
		t.Log(diff)
		t.Error("Did not aggregate attestations")
	}
}

func TestAggregateAttestations_MultipleAttestationsDifferentRoots(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}
	mockRoot := [32]byte{}
	d := &ethpb.AttestationData{
		BeaconBlockRoot: mockRoot[:],
		Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
		Target:          &ethpb.Checkpoint{Root: mockRoot[:]},
	}
	d1, ok := proto.Clone(d).(*ethpb.AttestationData)
	if !ok {
		t.Fatal("Entity is not of type *ethpb.AttestationData")
	}
	d1.Slot = 1
	d2, ok := proto.Clone(d).(*ethpb.AttestationData)
	if !ok {
		t.Fatal("Entity is not of type *ethpb.AttestationData")
	}
	d2.Slot = 2

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)

	atts := []*ethpb.Attestation{
		{Data: d, AggregationBits: bitfield.Bitlist{0b100001}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b100010}, Signature: sig.Marshal()},
		{Data: d1, AggregationBits: bitfield.Bitlist{0b100001}, Signature: sig.Marshal()},
		{Data: d1, AggregationBits: bitfield.Bitlist{0b100110}, Signature: sig.Marshal()},
		{Data: d2, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
	}

	if err := s.aggregateAttestations(context.Background(), atts); err != nil {
		t.Fatal(err)
	}

	if len(s.pool.UnaggregatedAttestations()) != 0 {
		t.Error("Unaggregated att pool did not clean up")
	}

	received := s.pool.AggregatedAttestations()
	sort.Slice(received, func(i, j int) bool {
		return received[i].Data.Slot < received[j].Data.Slot
	})
	att1, err := helpers.AggregateAttestations([]*ethpb.Attestation{atts[0], atts[1]})
	if err != nil {
		t.Error(err)
	}
	att2, err := helpers.AggregateAttestations([]*ethpb.Attestation{atts[2], atts[3]})
	if err != nil {
		t.Error(err)
	}
	wanted := append(att1, att2...)
	if !reflect.DeepEqual(wanted, received) {
		t.Error("Did not aggregate attestations")
	}
}
