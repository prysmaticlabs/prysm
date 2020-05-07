package bls_test

import (
	"testing"

	bls2 "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func BenchmarkPairing(b *testing.B) {
	if err := bls2.Init(bls2.BLS12_381); err != nil {
		b.Fatal(err)
	}
	if err := bls2.SetETHmode(bls2.EthModeDraft05); err != nil {
		panic(err)
	}
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
	sig := sk.Sign(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !sig.Verify(msg, sk.PublicKey()) {
			b.Fatal("could not verify sig")
		}
	}
}

func BenchmarkSignature_AggregateVerify(b *testing.B) {
	sigN := 128 // MAX_ATTESTATIONS per block.

	var pks []*bls.PublicKey
	var sigs []*bls.Signature
	var msgs [][32]byte
	for i := 0; i < sigN; i++ {
		msg := [32]byte{'s', 'i', 'g', 'n', 'e', 'd', byte(i)}
		sk := bls.RandKey()
		sig := sk.Sign(msg[:])
		pks = append(pks, sk.PublicKey())
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	aggregated := bls.Aggregate(sigs)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !aggregated.AggregateVerify(pks, msgs) {
			b.Fatal("could not verify aggregate sig")
		}
	}
}

func BenchmarkSecretKey_Marshal(b *testing.B) {
	key := bls.RandKey()
	d := key.Marshal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bls.SecretKeyFromBytes(d)
		_ = err
	}
}
