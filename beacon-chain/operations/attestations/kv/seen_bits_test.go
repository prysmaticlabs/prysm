package kv

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestAttCaches_hasSeenBit(t *testing.T) {
	c := NewAttCaches()
	d := &ethpb.AttestationData{
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		BeaconBlockRoot: make([]byte, 32),
	}
	seenA1 := &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000011}, Signature: make([]byte, 96)}
	seenA2 := &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b11100000}, Signature: make([]byte, 96)}
	require.NoError(t, c.insertSeenBit(seenA1))
	require.NoError(t, c.insertSeenBit(seenA2))
	tests := []struct {
		att  *ethpb.Attestation
		want bool
	}{
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000000}, Signature: make([]byte, 96)}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000001}, Signature: make([]byte, 96)}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b11100000}, Signature: make([]byte, 96)}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000011}, Signature: make([]byte, 96)}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b00001000}, Signature: make([]byte, 96)}, want: false},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b11110111}, Signature: make([]byte, 96)}, want: false},
	}
	for _, tt := range tests {
		got, err := c.hasSeenBit(tt.att)
		require.NoError(t, err)
		if got != tt.want {
			t.Errorf("hasSeenBit() got = %v, want %v", got, tt.want)
		}
	}
}
