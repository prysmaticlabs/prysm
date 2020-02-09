package bls_test

import (
	"testing"

	bls2 "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func BenchmarkPairing(b *testing.B) {
	bls2.Init(bls2.BLS12_381)
	newGt := &bls2.GT{}
	newG1 := &bls2.G1{}
	newG2 := &bls2.G2{}

	newGt.SetInt64(10)
	hash := hashutil.Hash([]byte{})
	err := newG1.HashAndMapTo(hash[:])
	if err != nil {
		b.Fatal(err)
	}
	err = newG2.HashAndMapTo(hash[:])
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bls2.Pairing(newGt, newG1, newG2)
	}

}
func BenchmarkSignature_Verify(b *testing.B) {
	sk := bls.RandKey()

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
	msg := [32]byte{'s', 'i', 'g', 'n', 'e', 'd'}
	domain := uint64(0)

	var aggregated *bls.Signature
	var pks []*bls.PublicKey
	for i := 0; i < sigN; i++ {
		sk := bls.RandKey()
		sig := sk.Sign(msg[:], domain)
		if aggregated == nil {
			aggregated = bls.AggregateSignatures([]*bls.Signature{sig})
		} else {
			aggregated = bls.AggregateSignatures([]*bls.Signature{aggregated, sig})
		}
		pks = append(pks, sk.PublicKey())
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !aggregated.VerifyAggregateCommon(pks, msg, domain) {
			b.Fatal("could not verify aggregate sig")
		}
	}
}
