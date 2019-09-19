package bls_test

import (
	"crypto/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

func BenchmarkSignature_Verify(b *testing.B) {
	sk, err := bls.RandKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	msg := []byte("Some msg")
	domain := uint64(42)
	sig := sk.Sign(msg, domain)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !sig.Verify(msg, sk.PublicKey(), domain) {
			b.Fatal("could not verify sig")
		}
	}
}

func BenchmarkSignature_VerifyAggregate(b *testing.B) {
	sigN := 128 // MAX_ATTESTATIONS per block.
	msg := []byte("signed message")
	domain := uint64(0)

	var aggregated *bls.Signature
	var pks []*bls.PublicKey
	for i := 0; i < sigN; i++ {
		sk, err := bls.RandKey(rand.Reader)
		if err != nil {
			b.Fatal(err)
		}
		sig := sk.Sign(msg, domain)
		aggregated = bls.AggregateSignatures([]*bls.Signature{aggregated, sig})
		pks = append(pks, sk.PublicKey())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !aggregated.VerifyAggregateCommon(pks, msg, domain) {
			b.Fatal("could not verify aggregate sig")
		}
	}
}
