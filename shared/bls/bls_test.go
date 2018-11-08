package bls

import (
	"testing"
)

func TestSign(t *testing.T) {
	sk := &SecretKey{}
	msg := []byte{}
	if _, err := Sign(sk, msg); err != nil {
		t.Errorf("Expected nil error, received %v", err)
	}
}

func TestPublicKey(t *testing.T) {
	sk := &SecretKey{}
	if _, err := sk.PublicKey(); err != nil {
		t.Errorf("Expected nil error, received %v", err)
	}
}

func TestVerifySig(t *testing.T) {
	pk := &PublicKey{}
	msg := []byte{}
	sig := &Signature{}
	if _, err := VerifySig(pk, msg, sig); err != nil {
		t.Errorf("Expected nil error, received %v", err)
	}
}

func TestVerifyAggregateSig(t *testing.T) {
	pk := &PublicKey{}
	msg := []byte{}
	asig := &Signature{}
	if _, err := VerifyAggregateSig([]*PublicKey{pk}, msg, asig); err != nil {
		t.Errorf("Expected nil error, received %v", err)
	}
}

func TestBatchVerify(t *testing.T) {
	pk := &PublicKey{}
	msg := []byte{}
	sig := &Signature{}
	if _, err := BatchVerify([]*PublicKey{pk}, msg, []*Signature{sig}); err != nil {
		t.Errorf("Expected nil error, received %v", err)
	}
}

func TestAggregateSigs(t *testing.T) {
	sig := &Signature{}
	if _, err := AggregateSigs([]*Signature{sig}); err != nil {
		t.Errorf("Expected nil error, received %v", err)
	}
}
