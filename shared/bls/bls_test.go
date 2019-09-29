package bls_test

import (
	"crypto/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestVerifySignature(t *testing.T) {
	msg := make([]byte, 32)
	var domain uint64 = 1
	secretKey, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := secretKey.PublicKey()
	signature := secretKey.Sign(msg, domain)
	if !signature.Verify(msg, pubKey, domain) {
		t.Fatal("verification fails")
	}
}

func TestVerifySignatureAggregatedCommon(t *testing.T) {
	msg := make([]byte, 32)
	var domain uint64 = 1
	signerSize := 10
	pubkeys := make([]*bls.PublicKey, signerSize)
	signatures := make([]*bls.Signature, signerSize)
	for i := 0; i < signerSize; i++ {
		secretKey, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pubkeys[i] = secretKey.PublicKey()
		signatures[i] = secretKey.Sign(msg, domain)
	}
	signature := bls.AggregateSignatures(signatures)
	if !signature.VerifyAggregateCommon(pubkeys, msg, domain) {
		t.Fatal("verification fails")
	}
}

func TestVerifySignatureAggregated(t *testing.T) {
	msgs := make([][32]byte, 10)
	var domain uint64 = 1
	signerSize := 10
	pubkeys := make([]*bls.PublicKey, signerSize)
	signatures := make([]*bls.Signature, signerSize)
	for i := 0; i < signerSize; i++ {
		msgs[i][0] = byte(i)
		secretKey, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		pubkeys[i] = secretKey.PublicKey()
		signatures[i] = secretKey.Sign(msgs[i][:], domain)
	}
	signature := bls.AggregateSignatures(signatures)
	if !signature.VerifyAggregate(pubkeys, msgs, domain) {
		t.Fatal("verification fails")
	}
}
