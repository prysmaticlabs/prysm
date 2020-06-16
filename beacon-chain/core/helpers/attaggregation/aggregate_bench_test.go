package attaggregation

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func BenchmarkAggregate_Aggregate(b *testing.B) {
	// Each test defines the aggregation bitfield inputs and the wanted output result.
	tests := []struct {
		name   string
		inputs []bitfield.Bitlist
		want   []bitfield.Bitlist
	}{
		{
			name:   "64 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(64),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "64 attestations with 8 random bits set",
			inputs: bitlistsWithMultipleBitSet(64, 8),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "64 attestations with 16 random bits set",
			inputs: bitlistsWithMultipleBitSet(64, 16),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "64 attestations with 32 random bits set",
			inputs: bitlistsWithMultipleBitSet(64, 32),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "128 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(128),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(128),
			},
		},
		{
			name:   "256 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(256),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(256),
			},
		},
		{
			name:   "512 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(512),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(512),
			},
		},
		{
			name:   "1024 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(1024),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(1024),
			},
		},
	}

	var makeAttestationsFromBitlists = func(bl []bitfield.Bitlist) []*ethpb.Attestation {
		atts := make([]*ethpb.Attestation, len(bl))
		for i, b := range bl {
			atts[i] = &ethpb.Attestation{
				AggregationBits: b,
				Data:            nil,
				Signature:       bls.NewAggregateSignature().Marshal(),
			}
		}
		return atts
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			atts := makeAttestationsFromBitlists(tt.inputs)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Aggregate(atts)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
