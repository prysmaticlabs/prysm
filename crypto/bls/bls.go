// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum.
package bls

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/crypto/bls/herumi"
)

// Initialize herumi temporarily while we transition to blst for ethdo.
func init() {
	herumi.HerumiInit()
}

// SecretKeyFromBytes creates a BLS private key from a BigEndian byte slice.
func SecretKeyFromBytes(privKey []byte) (SecretKey, error) {
	return blst.SecretKeyFromBytes(privKey)
}

// SecretKeyFromBigNum takes in a big number string and creates a BLS private key.
func SecretKeyFromBigNum(s string) (SecretKey, error) {
	num := new(big.Int)
	num, ok := num.SetString(s, 10)
	if !ok {
		return nil, errors.New("could not set big int from string")
	}
	bts := num.Bytes()
	if len(bts) != 32 {
		return nil, errors.Errorf("provided big number string sets to a key unequal to 32 bytes: %d != 32", len(bts))
	}
	return SecretKeyFromBytes(bts)
}

// PublicKeyFromBytes creates a BLS public key from a  BigEndian byte slice.
func PublicKeyFromBytes(pubKey []byte) (PublicKey, error) {
	return blst.PublicKeyFromBytes(pubKey)
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (Signature, error) {
	return blst.SignatureFromBytes(sig)
}

// AggregatePublicKeys aggregates the provided raw public keys into a single key.
func AggregatePublicKeys(pubs [][]byte) (PublicKey, error) {
	return blst.AggregatePublicKeys(pubs)
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []common.Signature) common.Signature {
	return blst.AggregateSignatures(sigs)
}

// VerifyMultipleSignatures verifies multiple signatures for distinct messages securely.
func VerifyMultipleSignatures(sigs [][]byte, msgs [][32]byte, pubKeys []common.PublicKey) (bool, error) {
	return blst.VerifyMultipleSignatures(sigs, msgs, pubKeys)
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() common.Signature {
	return blst.NewAggregateSignature()
}

// RandKey creates a new private key using a random input.
func RandKey() (common.SecretKey, error) {
	return blst.RandKey()
}
