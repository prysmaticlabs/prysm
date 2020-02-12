package attestations

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
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
	var mockRoot [32]byte

	unaggregatedAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{
			Slot:            2,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b101000}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            0,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100010}, Signature: sig.Marshal()},
	}
	aggregatedAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{
			Slot:            2,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b111000}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100011}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            0,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b110001}, Signature: sig.Marshal()},
	}
	blockAtts := []*ethpb.Attestation{
		{Data: &ethpb.AttestationData{
			Slot:            2,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100001}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            0,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
		{Data: &ethpb.AttestationData{
			Slot:            2,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b111000}, Signature: sig.Marshal()}, // Duplicated
		{Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: mockRoot[:],
			Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
			Target:          &ethpb.Checkpoint{Root: mockRoot[:]}}, AggregationBits: bitfield.Bitlist{0b100011}, Signature: sig.Marshal()}, // Duplicated
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
	mockRoot := [32]byte{}
	d := &ethpb.AttestationData{
		BeaconBlockRoot: mockRoot[:],
		Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
		Target:          &ethpb.Checkpoint{Root: mockRoot[:]},
	}

	unaggregatedAtts := []*ethpb.Attestation{
		{Data: d, AggregationBits: bitfield.Bitlist{0b101000}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b100100}, Signature: sig.Marshal()},
	}
	aggregatedAtts := []*ethpb.Attestation{
		{Data: d, AggregationBits: bitfield.Bitlist{0b101100}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b110010}, Signature: sig.Marshal()},
	}
	blockAtts := []*ethpb.Attestation{
		{Data: d, AggregationBits: bitfield.Bitlist{0b110010}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b100010}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b110010}, Signature: sig.Marshal()}, // Duplicated
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
		t.Error("Did not aggregate and save for batch")
	}
}

func TestAggregateAndSaveForkChoiceAtts_Single(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"), 0 /*domain*/)
	mockRoot := [32]byte{}
	d := &ethpb.AttestationData{
		BeaconBlockRoot: mockRoot[:],
		Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
		Target:          &ethpb.Checkpoint{Root: mockRoot[:]},
	}

	atts := []*ethpb.Attestation{
		{Data: d, AggregationBits: bitfield.Bitlist{0b101}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b110}, Signature: sig.Marshal()}}
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
	mockRoot := [32]byte{}
	d := &ethpb.AttestationData{
		BeaconBlockRoot: mockRoot[:],
		Source:          &ethpb.Checkpoint{Root: mockRoot[:]},
		Target:          &ethpb.Checkpoint{Root: mockRoot[:]},
	}
	d1 := proto.Clone(d).(*ethpb.AttestationData)
	d1.Slot = 1
	d2 := proto.Clone(d).(*ethpb.AttestationData)
	d2.Slot = 2

	atts1 := []*ethpb.Attestation{
		{Data: d, AggregationBits: bitfield.Bitlist{0b101}, Signature: sig.Marshal()},
		{Data: d, AggregationBits: bitfield.Bitlist{0b110}, Signature: sig.Marshal()},
	}
	if err := s.aggregateAndSaveForkChoiceAtts(atts1); err != nil {
		t.Fatal(err)
	}
	atts2 := []*ethpb.Attestation{
		{Data: d1, AggregationBits: bitfield.Bitlist{0b10110}, Signature: sig.Marshal()},
		{Data: d1, AggregationBits: bitfield.Bitlist{0b11100}, Signature: sig.Marshal()},
		{Data: d1, AggregationBits: bitfield.Bitlist{0b11000}, Signature: sig.Marshal()},
	}
	if err := s.aggregateAndSaveForkChoiceAtts(atts2); err != nil {
		t.Fatal(err)
	}
	att3 := []*ethpb.Attestation{
		{Data: d2, AggregationBits: bitfield.Bitlist{0b1100}, Signature: sig.Marshal()},
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

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, Signature: []byte{'A'}, AggregationBits: bitfield.Bitlist{0x13} /* 0b00010011 */}
	got, err := s.seen(att1)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Wanted false, got true")
	}

	time.Sleep(100 * time.Millisecond)

	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, Signature: []byte{'A'}, AggregationBits: bitfield.Bitlist{0x17} /* 0b00010111 */}
	got, err = s.seen(att2)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("Wanted false, got true")
	}

	time.Sleep(100 * time.Millisecond)

	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, Signature: []byte{'A'}, AggregationBits: bitfield.Bitlist{0x17} /* 0b00010111 */}
	got, err = s.seen(att3)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("Wanted true, got false")
	}
}

func TestService_seen(t *testing.T) {
	// Attestation are checked in order of this list.
	tests := []struct {
		att  *ethpb.Attestation
		want bool
	}{
		{
			att: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b11011},
				Data:            &ethpb.AttestationData{Slot: 1},
			},
			want: false,
		},
		{
			att: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b11011},
				Data:            &ethpb.AttestationData{Slot: 1},
			},
			want: true, // Exact same attestation should return true
		},
		{
			att: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b10101},
				Data:            &ethpb.AttestationData{Slot: 1},
			},
			want: false, // Haven't seen the bit at index 2 yet.
		},
		{
			att: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b11111},
				Data:            &ethpb.AttestationData{Slot: 1},
			},
			want: true, // We've full committee at this point.
		},
		{
			att: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b11111},
				Data:            &ethpb.AttestationData{Slot: 2},
			},
			want: false, // Different root is different bitlist.
		},
		{
			att: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b11111001},
				Data:            &ethpb.AttestationData{Slot: 1},
			},
			want: false, // Sanity test that an attestation of different lengths does not panic.
		},
	}

	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	if err != nil {
		t.Fatal(err)
	}

	for i, tt := range tests {
		if got, _ := s.seen(tt.att); got != tt.want {
			t.Errorf("Test %d failed. Got=%v want=%v", i, got, tt.want)
		}
		time.Sleep(10) // Sleep briefly for cache to routine to buffer.
	}
}
