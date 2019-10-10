// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/herumi/bls-go-binary/bls"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var pubkeyCache = ccache.New(ccache.Configure())

// CurveOrder for the BLS12-381 curve.
const CurveOrder = "52435875175126190479447740508185965837690552500527637822603658699938581184513"

var curveOrder, _ = new(big.Int).SetString(CurveOrder, 10)

// RandKey creates a new private key using a random method provided as an io.Reader.
func RandKey() *bls.SecretKey {
	secKey := &bls.SecretKey{}
	secKey.SetByCSPRNG()
	return secKey
}

// SecretKeyFromBytes creates a BLS private key from a LittleEndian byte slice.
func SecretKeyFromBytes(priv []byte) (*bls.SecretKey, error) {
	secKey := &bls.SecretKey{}
	if err := secKey.Deserialize(priv); err != nil {
		return nil, err
	}
	return secKey, nil
}

// PublicKeyFromBytes creates a BLS public key from a  LittleEndian byte slice.
func PublicKeyFromBytes(pub []byte) (*bls.PublicKey, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &bls.PublicKey{}, nil
	}
	cv := pubkeyCache.Get(string(pub))
	if cv != nil && cv.Value() != nil && featureconfig.Get().EnableBLSPubkeyCache {
		return cv.Value().(*bls.PublicKey), nil
	}
	pubkey := &bls.PublicKey{}
	if err := pubkey.Deserialize(pub); err != nil {
		return pubkey, nil
	}
	pubkeyCache.Set(string(pub), pubkey, 48*time.Hour)
	return pubkey, nil
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (*bls.Sign, error) {
	if featureconfig.Get().SkipBLSVerify {
		return &bls.Sign{}, nil
	}
	s := &bls.Sign{}
	if err := s.Deserialize(sig); err != nil {
		return nil, err
	}
	return s, nil
}

// Verify a bls signature given a public key, a message, and a domain.
func VerifyWithDomain(signature *bls.Sign, msg [32]byte, pub *bls.PublicKey, domain uint64) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], domain)

	return signature.VerifyHash(pub, HashWithDomain(msg, b))

}

// VerifyAggregate verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func VerifyAggregate(signature *bls.Sign, pubKeys []*bls.PublicKey, msg [][32]byte, domain uint64) bool {
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
	messageHashes := make([][]byte, len(msg))
	deRefPubkeys := make([]bls.PublicKey, len(pubKeys))

	for i, singleMsg := range msg {
		messageHashes[i] = HashWithDomain(singleMsg, b)
		deRefPubkeys[i] = *pubKeys[i]
	}

	return signature.VerifyAggregateHashes(deRefPubkeys, messageHashes)
}

// VerifyAggregateCommon verifies each public key against its respective message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func VerifyAggregateCommon(signature *bls.Sign, pubKeys []*bls.PublicKey, msg []byte, domain uint64) bool {
	if featureconfig.Get().SkipBLSVerify {
		return true
	}
	if len(pubKeys) == 0 {
		return false
	}
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], domain)

	aggPubkey := AggregatePubkeys(pubKeys)

	return signature.VerifyHash(aggPubkey, HashWithDomain(bytesutil.ToBytes32(msg), b))
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []*bls.Sign) *bls.Sign {
	if featureconfig.Get().SkipBLSVerify {
		return sigs[0]
	}
	firstSig := sigs[0]
	for i := 1; i < len(sigs); i++ {
		firstSig.Add(sigs[i])
	}
	return firstSig
}

func AggregatePubkeys(pubkeys []*bls.PublicKey) *bls.PublicKey {
	if featureconfig.Get().SkipBLSVerify {
		return pubkeys[0]
	}
	firstKey := pubkeys[0]
	for i := 1; i < len(pubkeys); i++ {
		firstKey.Add(pubkeys[i])
	}
	return firstKey
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
