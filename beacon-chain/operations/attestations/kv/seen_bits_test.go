package kv

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestAttCaches_hasSeenBit(t *testing.T) {
	c := NewAttCaches()
	d := &ethpb.AttestationData{}
	seenA1 := &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000011}}
	seenA2 := &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b11100000}}
	require.NoError(t, c.insertSeenBit(seenA1))
	require.NoError(t, c.insertSeenBit(seenA2))
	tests := []struct {
		att  *ethpb.Attestation
		want bool
	}{
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000000}}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000001}}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b11100000}}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b10000011}}, want: true},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b00001000}}, want: false},
		{att: &ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b11110111}}, want: false},
	}
	for _, tt := range tests {
		got, err := c.hasSeenBit(tt.att)
		require.NoError(t, err)
		if got != tt.want {
			t.Errorf("hasSeenBit() got = %v, want %v", got, tt.want)
		}
	}
}
