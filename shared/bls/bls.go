// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"fmt"

	"github.com/dgraph-io/ristretto"
	bls12 "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	if err := bls12.Init(bls12.BLS12_381); err != nil {
		panic(err)
	}
	if err := bls12.SetETHmode(1); err != nil {
		panic(err)
	}
}

// DomainByteLength length of domain byte array.
const DomainByteLength = 4

var maxKeys = int64(100000)
var pubkeyCache, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: maxKeys,
	MaxCost:     1 << 19, // 500 kb is cache max size
	BufferItems: 64,
})

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

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
	if cv, ok := pubkeyCache.Get(string(pub)); ok {
		return cv.(*PublicKey).Copy()
	}
	pubKey := &bls12.PublicKey{}
	err := pubKey.Deserialize(pub)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into public key")
	}
	pubKeyObj := &PublicKey{p: pubKey}
	copiedKey, err := pubKeyObj.Copy()
	if err != nil {
		return nil, errors.Wrap(err, "could not copy public key")
	}
	pubkeyCache.Set(string(pub), copiedKey, 48)
	return pubKeyObj, nil
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

// Sign a message using a secret key - in a beacon/validator client.
func (s *SecretKey) Sign(msg []byte) *Signature {
	if featureconfig.Get().SkipBLSVerify {
		return &Signature{}
	}
	signature := s.p.SignByte(msg)
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
	return p.p.Serialize()
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

// Verify a bls signature given a public key, a message.
func (s *Signature) Verify(msg []byte, pub *PublicKey) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	return s.s.VerifyByte(pub.p, msg)
}

// VerifyAggregate verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregate(pubKeys []*PublicKey, msg [][32]byte) bool {
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
	hashes := make([][]byte, 0, len(msg))
	var rawKeys []bls12.PublicKey
	for i := 0; i < size; i++ {
		hashes = append(hashes, msg[i][:])
		rawKeys = append(rawKeys, *pubKeys[i].p)
	}
	return s.s.VerifyAggregateHashes(rawKeys, hashes)
}

// AggregateVerify verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) AggregateVerify(pubKeys []*PublicKey, msgs [][32]byte) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	size := len(pubKeys)
	if size == 0 {
		return false
	}
	if size != len(msgs) {
		return false
	}
	msgSlices := []byte{}
	var rawKeys []bls12.PublicKey
	for i := 0; i < size; i++ {
		msgSlices = append(msgSlices, msgs[i][:]...)
		rawKeys = append(rawKeys, *pubKeys[i].p)
	}
	return s.s.AggregateVerify(rawKeys, msgSlices)
}

// FastAggregateVerify verifies all the provided pubkeys with their aggregated signature.
func (s *Signature) FastAggregateVerify(pubKeys []*PublicKey, msg [32]byte) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	if len(pubKeys) == 0 {
		return false
	}
	rawKeys := make([]bls12.PublicKey, len(pubKeys))
	for i := 0; i < len(pubKeys); i++ {
		rawKeys[i] = *pubKeys[i].p
	}

	return s.s.FastAggregateVerify(rawKeys, msg[:])
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

	// Copy signature
	signature := *sigs[0].s
	for i := 1; i < len(sigs); i++ {
		signature.Add(sigs[i].s)
	}
	return &Signature{s: &signature}
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	if featureconfig.Get().SkipBLSVerify {
		return make([]byte, params.BeaconConfig().BLSSignatureLength)
	}

	return s.s.Serialize()
}
