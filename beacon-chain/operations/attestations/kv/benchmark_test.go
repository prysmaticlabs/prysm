package kv_test

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations/kv"
)

func BenchmarkAttCaches(b *testing.B) {
	ac := kv.NewAttCaches()

	att := &ethpb.Attestation{}

	for i := 0; i < b.N; i++ {
		if err := ac.SaveUnaggregatedAttestation(att); err != nil {
			b.Error(err)
		}
		if err := ac.DeleteAggregatedAttestation(att); err != nil {
			b.Error(err)
		}
	}
}
