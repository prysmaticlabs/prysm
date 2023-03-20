package kv_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations/kv"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
)

func BenchmarkAttCaches(b *testing.B) {
	ac := kv.NewAttCaches()

	att := &ethpb.Attestation{}

	for i := 0; i < b.N; i++ {
		assert.NoError(b, ac.SaveUnaggregatedAttestation(att))
		assert.NoError(b, ac.DeleteAggregatedAttestation(att))
	}
}
