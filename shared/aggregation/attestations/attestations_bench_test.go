package attestations

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	aggtesting "github.com/prysmaticlabs/prysm/shared/aggregation/testing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func BenchmarkAggregateAttestations_Aggregate(b *testing.B) {
	// Override expensive BLS aggregation method with cheap no-op such that this benchmark profiles
	// the logic of aggregation selection rather than BLS logic.
	aggregateSignatures = func(sigs []common.Signature) common.Signature {
		return sigs[0]
	}
	signatureFromBytes = func(sig []byte) (common.Signature, error) {
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
	}{
		{
			name:   "64 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(64, bitlistLen),
		},
		{
			name:   "64 attestations with 8 random bits set",
			inputs: aggtesting.BitlistsWithMultipleBitSet(b, 64, bitlistLen, 8),
		},
		{
			name:   "64 attestations with 16 random bits set",
			inputs: aggtesting.BitlistsWithMultipleBitSet(b, 64, bitlistLen, 16),
		},
		{
			name:   "64 attestations with 32 random bits set",
			inputs: aggtesting.BitlistsWithMultipleBitSet(b, 64, bitlistLen, 32),
		},
		{
			name:   "256 attestations with 32 random bits set",
			inputs: aggtesting.BitlistsWithMultipleBitSet(b, 256, bitlistLen, 32),
		},
		{
			name:   "256 attestations with 64 random bits set",
			inputs: aggtesting.BitlistsWithMultipleBitSet(b, 256, bitlistLen, 64),
		},
		{
			name:   "128 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(128, bitlistLen),
		},
		{
			name:   "256 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(256, bitlistLen),
		},
		{
			name:   "512 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(512, bitlistLen),
		},
		{
			name:   "1024 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(1024, bitlistLen),
		},
	}

	b.Run("max-cover", func(b *testing.B) {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				atts := aggtesting.MakeAttestationsFromBitlists(tt.inputs)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := Aggregate(atts)
					require.NoError(b, err)
				}
			})
		}
	})

	b.Run("naive", func(b *testing.B) {
		resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
			AttestationAggregationStrategy: string(NaiveAggregation),
		})
		defer resetCfg()

		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				atts := aggtesting.MakeAttestationsFromBitlists(tt.inputs)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := Aggregate(atts)
					require.NoError(b, err)
				}
			})
		}
	})
}
