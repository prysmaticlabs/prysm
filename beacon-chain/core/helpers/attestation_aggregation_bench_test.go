package helpers

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func bitlistWithAllBitsSet(length uint64) bitfield.Bitlist {
	b := bitfield.NewBitlist(length)
	for i := uint64(0); i < length; i++ {
		b.SetBitAt(i, true)
	}
	return b
}

func bitlistsWithSingleBitSet(length uint64) []bitfield.Bitlist {
	lists := make([]bitfield.Bitlist, length)
	for i := uint64(0); i < length; i++ {
		b := bitfield.NewBitlist(length)
		b.SetBitAt(i, true)
		lists[i] = b
	}
	return lists
}

func BenchmarkAggregateAttestations(b *testing.B) {
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
				_, err := AggregateAttestations(atts)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
