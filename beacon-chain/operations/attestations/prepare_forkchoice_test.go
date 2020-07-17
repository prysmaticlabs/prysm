package attestations

import (
	"context"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBatchAttestations_Multiple(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"))
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

	wanted, err := attaggregation.Aggregate([]*ethpb.Attestation{aggregatedAtts[0], blockAtts[0]})
	require.NoError(t, err)
	aggregated, err := attaggregation.Aggregate([]*ethpb.Attestation{aggregatedAtts[1], blockAtts[1]})
	require.NoError(t, err)
	wanted = append(wanted, aggregated...)
	aggregated, err = attaggregation.Aggregate([]*ethpb.Attestation{aggregatedAtts[2], blockAtts[2]})
	require.NoError(t, err)

	wanted = append(wanted, aggregated...)
	if err := s.pool.AggregateUnaggregatedAttestations(); err != nil {
		return
	}
	received := s.pool.ForkchoiceAttestations()

	sort.Slice(received, func(i, j int) bool {
		return received[i].Data.Slot < received[j].Data.Slot
	})
	sort.Slice(wanted, func(i, j int) bool {
		return wanted[i].Data.Slot < wanted[j].Data.Slot
	})

	assert.DeepEqual(t, wanted, received)
}

func TestBatchAttestations_Single(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"))
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

	wanted, err := attaggregation.Aggregate(append(aggregatedAtts, unaggregatedAtts...))
	require.NoError(t, err)

	wanted, err = attaggregation.Aggregate(append(wanted, blockAtts...))
	require.NoError(t, err)

	got := s.pool.ForkchoiceAttestations()
	assert.DeepEqual(t, wanted, got)
}

func TestAggregateAndSaveForkChoiceAtts_Single(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"))
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

	wanted, err := attaggregation.Aggregate(atts)
	require.NoError(t, err)
	assert.DeepEqual(t, wanted, s.pool.ForkchoiceAttestations())
}

func TestAggregateAndSaveForkChoiceAtts_Multiple(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

	sk := bls.RandKey()
	sig := sk.Sign([]byte("dummy_test_data"))
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

	wanted, err := attaggregation.Aggregate(atts1)
	require.NoError(t, err)
	aggregated, err := attaggregation.Aggregate(atts2)
	require.NoError(t, err)

	wanted = append(wanted, aggregated...)
	wanted = append(wanted, att3...)

	received := s.pool.ForkchoiceAttestations()
	sort.Slice(received, func(i, j int) bool {
		return received[i].Data.Slot < received[j].Data.Slot
	})
	assert.DeepEqual(t, wanted, received)
}

func TestSeenAttestations_PresentInCache(t *testing.T) {
	s, err := NewService(context.Background(), &Config{Pool: NewPool()})
	require.NoError(t, err)

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, Signature: []byte{'A'}, AggregationBits: bitfield.Bitlist{0x13} /* 0b00010011 */}
	got, err := s.seen(att1)
	require.NoError(t, err)
	if got {
		t.Error("Wanted false, got true")
	}

	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, Signature: []byte{'A'}, AggregationBits: bitfield.Bitlist{0x17} /* 0b00010111 */}
	got, err = s.seen(att2)
	require.NoError(t, err)
	if got {
		t.Error("Wanted false, got true")
	}

	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{}, Signature: []byte{'A'}, AggregationBits: bitfield.Bitlist{0x17} /* 0b00010111 */}
	got, err = s.seen(att3)
	require.NoError(t, err)
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
	require.NoError(t, err)

	for _, tt := range tests {
		got, err := s.seen(tt.att)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}
