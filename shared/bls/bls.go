// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"fmt"
	gobls "github.com/phoreproject/bls"
	"io"
)

// Signature used in the BLS signature scheme.
type Signature struct {
	val *gobls.Signature
}

// SecretKey used in the BLS signature scheme.
type SecretKey struct {
    val *gobls.SecretKey
}

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	val *gobls.PublicKey
}

// RandKey creates a new private key using a random method provided as an io.Reader.
func RandKey(r io.Reader) (*SecretKey, error) {
	k, err := gobls.RandKey(r)
	if err != nil {
		return nil, fmt.Errorf("could not initialize secret key: %v", err)
	}
	return &SecretKey{val: k}, nil
}

// SecretKeyFromBytes creates a BLS private key from a byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	gobls.
	return &SecretKey{val: gobls.DeserializeSecretKey(priv)}, nil
}

// PublicKeyFromBytes creates a BLS public key from a byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	k, err := gobls.DeserializePublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal bytes into public key: %v", err)
	}
	return &PublicKey{val: k}, nil
}

func (s *SecretKey) PublicKey() *PublicKey {
	return &PublicKey{val: gobls.PrivToPub(s.val)}
}

// Sign a message using a secret key - in a beacon/validator client,
func (s *SecretKey) Sign(msg []byte, domain uint64) *Signature {
    sig := gobls.Sign(msg, s.val, domain)
    return &Signature{val: sig}
}

// Marshal a secret key into a byte slice.
func (s *SecretKey) Marshal() []byte {
	return s.val.Serialize()
}

// Marshal a public key into a byte slice.
func (p *PublicKey) Marshal() []byte {
	return p.val.Serialize()
}

// String fetches the string representation of a public key.
func (p *PublicKey) String() string {
	return p.val.String()
}

// Aggregate two public keys.
func (p *PublicKey) Aggregate(p2 *PublicKey) *PublicKey {
	p1 := p.val
	p1.Aggregate(p2.val)
	return &PublicKey{val: p1}
}

// Verify a bls signature given a public key, a message, and a domain.
func (p *PublicKey) Verify(msg []byte, sig *Signature, domain uint64) bool {
    return gobls.Verify(msg, p.val, sig.val, domain)
}

// VerifyAggregate verifies each public key against each message.
func (s *Signature) VerifyAggregate(pubKeys []*PublicKey, messages [][]byte, domain uint64) bool {
	var keys []*gobls.PublicKey
	for _, v := range pubKeys {
		keys = append(keys, v.val)
	}
	return s.val.VerifyAggregate(keys, messages, domain)
}

// VerifyAggregateCommon verifies each public key against a message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregateCommon(pubKeys []*PublicKey, msg []byte, domain uint64) bool {
	var keys []*gobls.PublicKey
	for _, v := range pubKeys {
		keys = append(keys, v.val)
	}
    return s.val.VerifyAggregateCommon(keys, msg, domain)
}

// Aggregate two signatures.
func (s *Signature) Aggregate(s2 *Signature) *Signature {
    s1 := s.val
    s1.Aggregate(s2.val)
	return &Signature{val: s1}
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []*Signature) *Signature {
	var ss []*gobls.Signature
	for _, v := range sigs {
		ss = append(ss, v.val)
	}
	return &Signature{val: gobls.AggregateSignatures(ss)}
}

// AggregatePublicKeys converts a list of public keys into a single, aggregated key.
func AggregatePublicKeys(pubKeys []*PublicKey) *PublicKey {
	var pks []*gobls.PublicKey
	for _, v := range pubKeys {
		pks = append(pks, v.val)
	}
	return &PublicKey{val: gobls.AggregatePublicKeys(pks)}
}
