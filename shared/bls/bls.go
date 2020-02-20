// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/ristretto"
	bls12 "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	err := bls12.Init(bls12.BLS12_381)
	if err != nil {
		panic(err)
	}
	bls12.SetETHserialization(true)
}

var maxKeys = int64(100000)
var pubkeyCache, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: maxKeys,
	MaxCost:     1 << 19, // 500 kb is cache max size
	BufferItems: 64,
})

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

// The size would be a combination of both the message(32 bytes) and domain(8 bytes) size.
const concatMsgDomainSize = 40

// Signature used in the BLS signature scheme.
type Signature struct {
	s *bls12.Sign
}

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	p *bls12.PublicKey
}

// SecretKey used in the BLS signature scheme.
type SecretKey struct {
	p *bls12.SecretKey
}

// RandKey creates a new private key using a random method provided as an io.Reader.
func RandKey() *SecretKey {
	secKey := &bls12.SecretKey{}
	secKey.SetByCSPRNG()
	return &SecretKey{secKey}
}

// SecretKeyFromBytes creates a BLS private key from a BigEndian byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	if len(priv) != params.BeaconConfig().BLSSecretKeyLength {
		return nil, fmt.Errorf("secret key must be %d bytes", params.BeaconConfig().BLSSecretKeyLength)
	}
	secKey := &bls12.SecretKey{}
	err := secKey.Deserialize(priv)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into secret key")
	}
	return &SecretKey{p: secKey}, err
}

// PublicKeyFromBytes creates a BLS public key from a  BigEndian byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &PublicKey{}, nil
	}
	if len(pub) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf("public key must be %d bytes", params.BeaconConfig().BLSPubkeyLength)
	}
	cv, ok := pubkeyCache.Get(string(pub))
	if ok {
		return cv.(*PublicKey).Copy()
	}
	pubKey := &bls12.PublicKey{}
	err := pubKey.Deserialize(pub)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into public key")
	}
	pubkeyObj := &PublicKey{p: pubKey}
	copiedKey, err := pubkeyObj.Copy()
	if err != nil {
		return nil, errors.Wrap(err, "could not copy pubkey")
	}
	pubkeyCache.Set(string(pub), copiedKey, 48)
	return pubkeyObj, nil
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &Signature{}, nil
	}
	if len(sig) != params.BeaconConfig().BLSSignatureLength {
		return nil, fmt.Errorf("signature must be %d bytes", params.BeaconConfig().BLSSignatureLength)
	}
	signature := &bls12.Sign{}
	err := signature.Deserialize(sig)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into signature")
	}
	return &Signature{s: signature}, nil
}

// PublicKey obtains the public key corresponding to the BLS secret key.
func (s *SecretKey) PublicKey() *PublicKey {
	return &PublicKey{p: s.p.GetPublicKey()}
}

func concatMsgAndDomain(msg []byte, domain uint64) []byte {
	b := [concatMsgDomainSize]byte{}
	binary.LittleEndian.PutUint64(b[32:], domain)
	copy(b[0:32], msg)
	return b[:]
}

// Sign a message using a secret key - in a beacon/validator client.
func (s *SecretKey) Sign(msg []byte, domain uint64) *Signature {
	if featureconfig.Get().SkipBLSVerify {
		return &Signature{}
	}
	signature := s.p.SignHashWithDomain(concatMsgAndDomain(msg, domain))
	return &Signature{s: signature}
}

// Marshal a secret key into a LittleEndian byte slice.
func (s *SecretKey) Marshal() []byte {
	keyBytes := s.p.Serialize()
	if len(keyBytes) < params.BeaconConfig().BLSSecretKeyLength {
		emptyBytes := make([]byte, params.BeaconConfig().BLSSecretKeyLength-len(keyBytes))
		keyBytes = append(emptyBytes, keyBytes...)
	}
	return keyBytes
}

// Marshal a public key into a LittleEndian byte slice.
func (p *PublicKey) Marshal() []byte {
	rawBytes := p.p.Serialize()
	return rawBytes
}

// Copy the public key to a new pointer reference.
func (p *PublicKey) Copy() (*PublicKey, error) {
	np := *p.p
	return &PublicKey{p: &np}, nil
}

// Aggregate two public keys.
func (p *PublicKey) Aggregate(p2 *PublicKey) *PublicKey {
	if featureconfig.Get().SkipBLSVerify {
		return p
	}
	p.p.Add(p2.p)
	return p
}

// Verify a bls signature given a public key, a message, and a domain.
func (s *Signature) Verify(msg []byte, pub *PublicKey, domain uint64) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	return s.s.VerifyHashWithDomain(pub.p, concatMsgAndDomain(msg, domain))
}

// VerifyAggregate verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregate(pubKeys []*PublicKey, msg [][32]byte, domain uint64) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	size := len(pubKeys)
	if size == 0 {
		return false
	}
	if size != len(msg) {
		return false
	}
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], domain)
	hashWithDomains := make([]byte, 0, size*concatMsgDomainSize)
	var rawKeys []bls12.PublicKey
	for i := 0; i < size; i++ {
		hashWithDomains = append(hashWithDomains, concatMsgAndDomain(msg[i][:], domain)...)
		rawKeys = append(rawKeys, *pubKeys[i].p)
	}
	return s.s.VerifyAggregateHashWithDomain(rawKeys, hashWithDomains)
}

// VerifyAggregateCommon verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregateCommon(pubKeys []*PublicKey, msg [32]byte, domain uint64) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	if len(pubKeys) == 0 {
		return false
	}
	//#nosec G104
	aggregated, _ := pubKeys[0].Copy()

	for i := 1; i < len(pubKeys); i++ {
		aggregated.p.Add(pubKeys[i].p)
	}

	return s.s.VerifyHashWithDomain(aggregated.p, concatMsgAndDomain(msg[:], domain))
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() *Signature {
	return &Signature{s: bls12.HashAndMapToSignature([]byte{'m', 'o', 'c', 'k'})}
}

// NewAggregatePubkey creates a blank public key.
func NewAggregatePubkey() *PublicKey {
	return &PublicKey{p: RandKey().PublicKey().p}
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []*Signature) *Signature {
	if len(sigs) == 0 {
		return nil
	}
	if featureconfig.Get().SkipBLSVerify {
		return sigs[0]
	}
	marshalled := sigs[0].s.Serialize()
	signature := &bls12.Sign{}
	//#nosec G104
	signature.Deserialize(marshalled)

	for i := 1; i < len(sigs); i++ {
		signature.Add(sigs[i].s)
	}
	return &Signature{s: signature}
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	if featureconfig.Get().SkipBLSVerify {
		return make([]byte, params.BeaconConfig().BLSSignatureLength)
	}

	rawBytes := s.s.Serialize()
	return rawBytes
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

// ComputeDomain returns the domain version for BLS private key to sign and verify with a zeroed 4-byte
// array as the fork version.
//
// def compute_domain(domain_type: DomainType, fork_version: Version=Version()) -> Domain:
//    """
//    Return the domain for the ``domain_type`` and ``fork_version``.
//    """
//    return Domain(domain_type + fork_version)
func ComputeDomain(domainType []byte) uint64 {
	return Domain(domainType, []byte{0, 0, 0, 0})
}

// HashWithDomain hashes 32 byte message and uint64 domain parameters a Fp2 element
func HashWithDomain(messageHash [32]byte, domain [8]byte) []byte {
	xReBytes := [41]byte{}
	xImBytes := [41]byte{}
	xBytes := make([]byte, 96)
	copy(xReBytes[:32], messageHash[:])
	copy(xReBytes[32:40], domain[:])
	copy(xReBytes[40:41], []byte{0x01})
	copy(xImBytes[:32], messageHash[:])
	copy(xImBytes[32:40], domain[:])
	copy(xImBytes[40:41], []byte{0x02})
	hashedxImBytes := hashutil.Hash(xImBytes[:])
	copy(xBytes[16:48], hashedxImBytes[:])
	hashedxReBytes := hashutil.Hash(xReBytes[:])
	copy(xBytes[64:], hashedxReBytes[:])
	return xBytes
}
