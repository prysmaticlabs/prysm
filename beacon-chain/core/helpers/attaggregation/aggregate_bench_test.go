package attaggregation

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func BenchmarkAggregate_Aggregate(b *testing.B) {
	// Override expensive BLS aggregation method with cheap no-op such that this benchmark profiles
	// the logic of aggregation selection rather than BLS logic.
	aggregateSignatures = func(sigs []*bls.Signature) *bls.Signature {
		return sigs[0]
	}
	signatureFromBytes = func(sig []byte) (*bls.Signature, error) {
		return bls.NewAggregateSignature(), nil
	}
	defer func() {
		aggregateSignatures = bls.AggregateSignatures
		signatureFromBytes = bls.SignatureFromBytes
	}()

	bitlistLen := params.BeaconConfig().MaxValidatorsPerCommittee

	tests := []struct {
		name   string
		inputs []bitfield.Bitlist
		want   []bitfield.Bitlist
	}{
		{
			name:   "64 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(64, bitlistLen),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "64 attestations with 8 random bits set",
			inputs: bitlistsWithMultipleBitSet(64, bitlistLen, 8),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "64 attestations with 16 random bits set",
			inputs: bitlistsWithMultipleBitSet(64, bitlistLen, 16),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "64 attestations with 32 random bits set",
			inputs: bitlistsWithMultipleBitSet(64, bitlistLen, 32),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(64),
			},
		},
		{
			name:   "128 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(128, bitlistLen),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(128),
			},
		},
		{
			name:   "256 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(256, bitlistLen),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(256),
			},
		},
		{
			name:   "512 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(512, bitlistLen),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(512),
			},
		},
		{
			name:   "1024 attestations with single bit set",
			inputs: bitlistsWithSingleBitSet(1024, bitlistLen),
			want: []bitfield.Bitlist{
				bitlistWithAllBitsSet(1024),
			},
		},
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
