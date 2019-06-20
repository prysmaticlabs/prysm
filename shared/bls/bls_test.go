package bls_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	bls2 "github.com/phoreproject/bls"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestMarshalUnmarshal(t *testing.T) {
	b := []byte("hi")
	b32 := bytesutil.ToBytes32(b)
	pk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	pk2, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pk.Marshal(), pk2.Marshal()) {
		t.Errorf("Keys not equal, received %#x == %#x", pk.Marshal(), pk2.Marshal())
	}
}

func TestSignVerify(t *testing.T) {
	priv, _ := bls.RandKey(rand.Reader)
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg, 0)
	if !sig.Verify(msg, pub, 0) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregate(t *testing.T) {
	pubkeys := make([]*bls.PublicKey, 0, 100)
	sigs := make([]*bls.Signature, 0, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv, _ := bls.RandKey(rand.Reader)
		pub := priv.PublicKey()
		sig := priv.Sign(msg, 0)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	aggSig := bls.AggregateSignatures(sigs)
	if !aggSig.VerifyAggregate(pubkeys, msg, 0) {
		t.Error("Signature did not verify")
	}
}

func TestHashG2WithDomain(t *testing.T) {
	var msg [32]byte
	copy(msg[:], []byte{'H', 'E', 'L', 'L', 'O'})
	domain := uint64(10)
	compHash := bls.HashG2WithDomainCompressed(msg, domain)
	unCompHash := bls.HashG2WithDomainUncompressed(msg, domain)
	g2Affine, err := bls2.DecompressG2(compHash)
	if err != nil {
		t.Fatal(err)
	}

	if unCompHash != g2Affine.SerializeBytes() {
		t.Errorf("Uncompressed and Compressed Hash do not point to the same affine point. Expected %#x but got %#x",
			unCompHash, g2Affine.SerializeBytes())
	}
}
