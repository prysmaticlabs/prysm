package bls_test

import (
	"bytes"
	"crypto/rand"
	"testing"

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
	var msgs [][32]byte
	for i := 0; i < 100; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i)}
		priv, _ := bls.RandKey(rand.Reader)
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:], 0)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	aggSig := bls.AggregateSignatures(sigs)
	if !aggSig.VerifyAggregate(pubkeys, msgs, 0) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregateCommon(t *testing.T) {
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
	if !aggSig.VerifyAggregateCommon(pubkeys, msg, 0) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregate_ReturnsFalseOnEmptyPubKeyList(t *testing.T) {
	var pubkeys []*bls.PublicKey
	sigs := make([]*bls.Signature, 0, 100)
	msg := []byte("hello")

	aggSig := bls.AggregateSignatures(sigs)
	if aggSig.VerifyAggregateCommon(pubkeys, msg, 0 /*domain*/) != false {
		t.Error("Expected VerifyAggregate to return false with empty input " +
			"of public keys.")
	}
}
