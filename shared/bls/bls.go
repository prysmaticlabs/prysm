// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"math/big"
	"time"

	"github.com/karlseguin/ccache"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"

	bls12 "github.com/kilic/bls12-381"
)

var pubkeyCache = ccache.New(ccache.Configure())

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

var curveOrder, _ = new(big.Int).SetString(CurveOrder, 10)

// Signature used in the BLS signature scheme.
type Signature struct {
	s *bls12.PointG2
}

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	p *bls12.PointG1
}

// SecretKey used in the BLS signature scheme.
type SecretKey struct {
	p *big.Int
}

// RandKey creates a new private key using a random method provided as an io.Reader.
func RandKey(r io.Reader) (*SecretKey, error) {
	k, err := rand.Int(r, curveOrder)
	if err != nil {
		return nil, err
	}
	return &SecretKey{k}, nil
}

// SecretKeyFromBytes creates a BLS private key from a LittleEndian byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	b := bytesutil.ToBytes32(priv)
	k := new(big.Int).SetBytes(b[:])
	if curveOrder.Cmp(k) < 0 {
		return nil, errors.New("invalid private key")
	}
	return &SecretKey{p: k}, nil
}

// PublicKeyFromBytes creates a BLS public key from a  LittleEndian byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return &PublicKey{}, nil
	}
	cv := pubkeyCache.Get(string(pub))
	if cv != nil && cv.Value() != nil && featureconfig.FeatureConfig().EnableBLSPubkeyCache {
		return cv.Value().(*PublicKey).Copy(), nil
	}
	b := bytesutil.ToBytes48(pub)
	g1Elems := bls12.NewG1(nil)
	p, err := g1Elems.FromCompressed(b[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into public key")
	}
	pubkey := &PublicKey{p: p}
	pubkeyCache.Set(string(pub), pubkey.Copy(), 48*time.Hour)
	return pubkey, nil
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return &Signature{}, nil
	}
	s, err := bls12.NewG2(nil).FromCompressed(sig)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into signature")
	}
	return &Signature{s: s}, nil
}

// PublicKey obtains the public key corresponding to the BLS secret key.
func (s *SecretKey) PublicKey() *PublicKey {
	p := &bls12.PointG1{}
	bls12.NewG1(nil).MulScalar(p, &bls12.G1One, s.p)
	return &PublicKey{p: p}
}

// Sign a message using a secret key - in a beacon/validator client.
func (s *SecretKey) Sign(msg []byte, domain uint64) *Signature {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return &Signature{}
	}
	g2 := bls12.NewG2(nil)
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], domain)
	signature := g2.MapToPoint(HashWithDomain(bytesutil.ToBytes32(msg), b))
	g2.MulScalar(signature, signature, s.p)
	return &Signature{s: signature}
}

// Marshal a secret key into a LittleEndian byte slice.
func (s *SecretKey) Marshal() []byte {
	keyBytes := s.p.Bytes()
	if len(keyBytes) < 32 {
		emptyBytes := make([]byte, 32-len(keyBytes))
		keyBytes = append(emptyBytes, keyBytes...)
	}
	return keyBytes
}

// Marshal a public key into a LittleEndian byte slice.
func (p *PublicKey) Marshal() []byte {
	return bls12.NewG1(nil).ToCompressed(p.p)
}

// Copy the public key to a new pointer reference.
func (p *PublicKey) Copy() *PublicKey {
	return &PublicKey{p: new(bls12.PointG1).Set(p.p)}
}

// Aggregate two public keys.
func (p *PublicKey) Aggregate(p2 *PublicKey) *PublicKey {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return p
	}
	bls12.NewG1(nil).Add(p.p, p.p, p2.p)
	return p
}

// Verify a bls signature given a public key, a message, and a domain.
func (s *Signature) Verify(msg []byte, pub *PublicKey, domain uint64) bool {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return true
	}
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], domain)
	e := bls12.NewBLSPairingEngine()
	target := &bls12.Fe12{}
	e.Pair(target,
		[]bls12.PointG1{
			bls12.G1NegativeOne,
			*pub.p,
		},
		[]bls12.PointG2{
			*s.s,
			*e.G2.MapToPoint(HashWithDomain(bytesutil.ToBytes32(msg), b)),
		},
	)
	return e.Fp12.Equal(&bls12.Fp12One, target)
}

// VerifyAggregate verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregate(pubKeys []*PublicKey, msg [][32]byte, domain uint64) bool {
	if featureconfig.FeatureConfig().SkipBLSVerify {
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
	points := make([]bls12.PointG1, size+1)
	e := bls12.NewBLSPairingEngine()
	e.G1.Copy(&points[0], &bls12.G1NegativeOne)
	twistPoints := make([]bls12.PointG2, size+1)
	e.G2.Copy(&twistPoints[0], s.s)
	for i := 0; i < size; i++ {
		e.G1.Copy(&points[i+1], pubKeys[i].p)
		e.G2.Copy(&twistPoints[i+1], e.G2.MapToPoint(HashWithDomain(msg[i], b)))
	}
	target := &bls12.Fe12{}
	e.Pair(target, points, twistPoints)
	return e.Fp12.Equal(&bls12.Fp12One, target)
}

// VerifyAggregateCommon verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregateCommon(pubKeys []*PublicKey, msg []byte, domain uint64) bool {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return true
	}
	if len(pubKeys) == 0 {
		return false
	}
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], domain)
	e := bls12.NewBLSPairingEngine()
	aggregated := &bls12.PointG1{}
	e.G1.Copy(aggregated, pubKeys[0].p)
	for i := 1; i < len(pubKeys); i++ {
		e.G1.Add(aggregated, aggregated, pubKeys[i].p)
	}
	target := &bls12.Fe12{}
	e.Pair(target,
		[]bls12.PointG1{
			bls12.G1NegativeOne,
			*aggregated,
		},
		[]bls12.PointG2{
			*s.s,
			*e.G2.MapToPoint(HashWithDomain(bytesutil.ToBytes32(msg), b)),
		},
	)
	return e.Fp12.Equal(&bls12.Fp12One, target)
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() *Signature {
	return &Signature{s: &bls12.PointG2{}}
}

// NewAggregatePubkey creates a blank public key.
func NewAggregatePubkey() *PublicKey {
	return &PublicKey{p: &bls12.PointG1{}}
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []*Signature) *Signature {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return sigs[0]
	}
	aggregated := NewAggregateSignature()
	g2 := bls12.NewG2(nil)
	for i := 0; i < len(sigs); i++ {
		sig := sigs[i]
		if sig == nil {
			continue
		}
		g2.Add(aggregated.s, aggregated.s, sig.s)
	}
	return aggregated
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	if featureconfig.FeatureConfig().SkipBLSVerify {
		return make([]byte, 96)
	}
	return bls12.NewG2(nil).ToCompressed(s.s)
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
