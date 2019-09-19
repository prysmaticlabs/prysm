// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"encoding/binary"
	"io"

	g1 "github.com/phoreproject/bls/g1pubs"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

// Signature used in the BLS signature scheme.
type Signature struct {
	val *g1.Signature
}

// SecretKey used in the BLS signature scheme.
type SecretKey struct {
	val *g1.SecretKey
}

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	val *g1.PublicKey
}

// RandKey creates a new private key using a random method provided as an io.Reader.
func RandKey(r io.Reader) (*SecretKey, error) {
	k, err := g1.RandKey(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize secret key")
	}
	return &SecretKey{val: k}, nil
}

// SecretKeyFromBytes creates a BLS private key from a LittleEndian byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	k := bytesutil.ToBytes32(priv)
	val := g1.DeserializeSecretKey(k)
	if val.GetFRElement() == nil {
		return nil, errors.New("invalid private key")
	}
	return &SecretKey{val}, nil
}

// PublicKeyFromBytes creates a BLS public key from a  LittleEndian byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	b := bytesutil.ToBytes48(pub)
	k, err := g1.DeserializePublicKey(b)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into public key")
	}
	return &PublicKey{val: k}, nil
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	b := bytesutil.ToBytes96(sig)
	s, err := g1.DeserializeSignature(b)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into signature")
	}
	return &Signature{val: s}, nil
}

// PublicKey obtains the public key corresponding to the BLS secret key.
func (s *SecretKey) PublicKey() *PublicKey {
	return &PublicKey{val: g1.PrivToPub(s.val)}
}

// Sign a message using a secret key - in a beacon/validator client,
func (s *SecretKey) Sign(msg []byte, domain uint64) *Signature {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, domain)
	sig := g1.SignWithDomain(bytesutil.ToBytes32(msg), s.val, bytesutil.ToBytes8(b))
	return &Signature{val: sig}
}

// Marshal a secret key into a LittleEndian byte slice.
func (s *SecretKey) Marshal() []byte {
	k := s.val.Serialize()
	return k[:]
}

// Marshal a public key into a LittleEndian byte slice.
func (p *PublicKey) Marshal() []byte {
	k := p.val.Serialize()
	return k[:]
}

// Aggregate two public keys.
func (p *PublicKey) Aggregate(p2 *PublicKey) *PublicKey {
	p1 := p.val
	p1.Aggregate(p2.val)
	return &PublicKey{val: p1}
}

// Verify a bls signature given a public key, a message, and a domain.
func (s *Signature) Verify(msg []byte, pub *PublicKey, domain uint64) bool {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, domain)
	return g1.VerifyWithDomain(bytesutil.ToBytes32(msg), pub.val, s.val, bytesutil.ToBytes8(b))
}

// VerifyAggregate verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregate(pubKeys []*PublicKey, msg [][32]byte, domain uint64) bool {
	if len(pubKeys) == 0 {
		return false // Otherwise panic in VerifyAggregateCommonWithDomain.
	}
	var keys []*g1.PublicKey
	for _, v := range pubKeys {
		keys = append(keys, v.val)
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, domain)
	return s.val.VerifyAggregateWithDomain(keys, msg, bytesutil.ToBytes8(b))
}

// VerifyAggregateCommon verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregateCommon(pubKeys []*PublicKey, msg []byte, domain uint64) bool {
	if len(pubKeys) == 0 {
		return false // Otherwise panic in VerifyAggregateCommonWithDomain.
	}
	var keys []*g1.PublicKey
	for _, v := range pubKeys {
		keys = append(keys, v.val)
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, domain)
	return s.val.VerifyAggregateCommonWithDomain(keys, bytesutil.ToBytes32(msg), bytesutil.ToBytes8(b))
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	k := s.val.Serialize()
	return k[:]
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []*Signature) *Signature {
	var ss []*g1.Signature
	for _, v := range sigs {
		if v == nil {
			continue
		}
		ss = append(ss, v.val)
	}
	return &Signature{val: g1.AggregateSignatures(ss)}
}

// Domain returns the bls domain given by the domain type and the operation 4 byte fork version.
//
// Spec pseudocode definition:
//  def get_domain(state: BeaconState, domain_type: DomainType, message_epoch: Epoch=None) -> Domain:
//    """
//    Return the signature domain (fork version concatenated with domain type) of a message.
//    """
//    epoch = get_current_epoch(state) if message_epoch is None else message_epoch
//    fork_version = state.fork.previous_version if epoch < state.fork.epoch else state.fork.current_version
//    return compute_domain(domain_type, fork_version)
func Domain(domainType []byte, forkVersion []byte) uint64 {
	b := []byte{}
	b = append(b, domainType[:4]...)
	b = append(b, forkVersion[:4]...)
	return bytesutil.FromBytes8(b)
}
