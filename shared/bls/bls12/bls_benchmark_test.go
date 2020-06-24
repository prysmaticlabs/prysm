package bls12_test

import (
	"testing"

	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/prysmaticlabs/prysm/shared/bls/bls12"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

func BenchmarkPairing(b *testing.B) {
	if err := bls.Init(bls.BLS12_381); err != nil {
		b.Fatal(err)
	}
	if err := bls.SetETHmode(bls.EthModeDraft07); err != nil {
		panic(err)
	}
	newGt := &bls.GT{}
	newG1 := &bls.G1{}
	newG2 := &bls.G2{}

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
		bls.Pairing(newGt, newG1, newG2)
	}

}
func BenchmarkSignature_Verify(b *testing.B) {
	sk := bls12.RandKey()

	msg := []byte("Some msg")
	sig := sk.Sign(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !sig.Verify(sk.PublicKey(), msg) {
			b.Fatal("could not verify sig")
		}
	}
}

func BenchmarkSignature_AggregateVerify(b *testing.B) {
	sigN := 128 // MAX_ATTESTATIONS per block.

	var pks []iface.PublicKey
	var sigs []iface.Signature
	var msgs [][32]byte
	for i := 0; i < sigN; i++ {
		msg := [32]byte{'s', 'i', 'g', 'n', 'e', 'd', byte(i)}
		sk := bls12.RandKey()
		sig := sk.Sign(msg[:])
		pks = append(pks, sk.PublicKey())
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	aggregated := bls12.Aggregate(sigs)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !aggregated.AggregateVerify(pks, msgs) {
			b.Fatal("could not verify aggregate sig")
		}
	}
}

func BenchmarkSecretKey_Marshal(b *testing.B) {
	key := bls12.RandKey()
	d := key.Marshal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bls12.SecretKeyFromBytes(d)
		_ = err
	}
}
