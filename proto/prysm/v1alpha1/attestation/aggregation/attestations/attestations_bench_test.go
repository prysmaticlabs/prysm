package attestations

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	aggtesting "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation/aggregation/testing"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func BenchmarkAggregateAttestations_Aggregate(b *testing.B) {
	// Override expensive BLS aggregation method with cheap no-op such that this benchmark profiles
	// the logic of aggregation selection rather than BLS logic.
	aggregateSignatures = func(sigs []bls.Signature) bls.Signature {
		return sigs[0]
	}
	signatureFromBytes = func(sig []byte) (bls.Signature, error) {
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
			name:   "256 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(256, bitlistLen),
		},
		{
			name:   "256 attestations with 64 random bits set",
			inputs: aggtesting.BitlistsWithSingleBitSet(256, bitlistLen),
		},
		{
			name:   "512 attestations with single bit set",
			inputs: aggtesting.BitlistsWithSingleBitSet(512, bitlistLen),
		},
		{
			name:   "1024 attestations with 64 random bits set",
			inputs: aggtesting.BitlistsWithMultipleBitSet(b, 1024, bitlistLen, 64),
		},
	}

	runner := func(atts []*ethpb.Attestation) {
		attsCopy := make([]*ethpb.Attestation, len(atts))
		for i, att := range atts {
			attsCopy[i] = ethpb.CopyAttestation(att)
		}
		_, err := Aggregate(attsCopy)
		require.NoError(b, err)
	}

	for _, tt := range tests {
		b.Run(fmt.Sprintf("naive_%s", tt.name), func(b *testing.B) {
			b.StopTimer()
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(NaiveAggregation),
			})
			atts := aggtesting.MakeAttestationsFromBitlists(tt.inputs)
			defer resetCfg()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				runner(atts)
			}
		})
		b.Run(fmt.Sprintf("max-cover_%s", tt.name), func(b *testing.B) {
			b.StopTimer()
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(MaxCoverAggregation),
			})
			atts := aggtesting.MakeAttestationsFromBitlists(tt.inputs)
			defer resetCfg()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				runner(atts)
			}
		})
		b.Run(fmt.Sprintf("opt-max-cover_%s", tt.name), func(b *testing.B) {
			b.StopTimer()
			resetCfg := features.InitWithReset(&features.Flags{
				AttestationAggregationStrategy: string(OptMaxCoverAggregation),
			})
			atts := aggtesting.MakeAttestationsFromBitlists(tt.inputs)
			defer resetCfg()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				runner(atts)
			}
		})
	}
}
