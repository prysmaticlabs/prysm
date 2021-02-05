// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls/blst"
	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/bls/herumi"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// SecretKeyFromBytes creates a BLS private key from a BigEndian byte slice.
func SecretKeyFromBytes(privKey []byte) (SecretKey, error) {
	if featureconfig.Get().EnableBlst {
		return blst.SecretKeyFromBytes(privKey)
	}
	return herumi.SecretKeyFromBytes(privKey)
}

// SecretKeyFromBigNum takes in a big number string and creates a BLS private key.
func SecretKeyFromBigNum(s string) (SecretKey, error) {
	num := new(big.Int)
	num, ok := num.SetString(s, 10)
	if !ok {
		return nil, errors.New("could not set big int from string")
	}
	bts := num.Bytes()
	// Pad key at the start with zero bytes to make it into a 32 byte key.
	if len(bts) < 32 {
		emptyBytes := make([]byte, 32-len(bts))
		bts = append(emptyBytes, bts...)
	}
	return SecretKeyFromBytes(bts)
}

// PublicKeyFromBytes creates a BLS public key from a  BigEndian byte slice.
func PublicKeyFromBytes(pubKey []byte) (PublicKey, error) {
	if featureconfig.Get().EnableBlst {
		return blst.PublicKeyFromBytes(pubKey)
	}
	return herumi.PublicKeyFromBytes(pubKey)
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (Signature, error) {
	if featureconfig.Get().EnableBlst {
		return blst.SignatureFromBytes(sig)
	}
	return herumi.SignatureFromBytes(sig)
}

// AggregatePublicKeys aggregates the provided raw public keys into a single key.
func AggregatePublicKeys(pubs [][]byte) (PublicKey, error) {
	if featureconfig.Get().EnableBlst {
		return blst.AggregatePublicKeys(pubs)
	}
	return herumi.AggregatePublicKeys(pubs)
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []common.Signature) common.Signature {
	if featureconfig.Get().EnableBlst {
		return blst.AggregateSignatures(sigs)
	}
	return herumi.AggregateSignatures(sigs)
}

// VerifyMultipleSignatures verifies multiple signatures for distinct messages securely.
func VerifyMultipleSignatures(sigs [][]byte, msgs [][32]byte, pubKeys []common.PublicKey) (bool, error) {
	if featureconfig.Get().EnableBlst {
		return blst.VerifyMultipleSignatures(sigs, msgs, pubKeys)
	}
	// Manually decompress each signature as herumi does not
	// have a batch decompress method.
	rawSigs := make([]Signature, len(sigs))
	var err error
	for i, s := range sigs {
		rawSigs[i], err = herumi.SignatureFromBytes(s)
		if err != nil {
			return false, err
		}
	}
	return herumi.VerifyMultipleSignatures(rawSigs, msgs, pubKeys)
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() common.Signature {
	if featureconfig.Get().EnableBlst {
		return blst.NewAggregateSignature()
	}
	return herumi.NewAggregateSignature()
}

// RandKey creates a new private key using a random input.
func RandKey() (common.SecretKey, error) {
	if featureconfig.Get().EnableBlst {
		return blst.RandKey()
	}
	return herumi.RandKey()
}
