package utils

import (
	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
)

// PublicKey is an interface for public keys
type PublicKey interface {
	Marshal() []byte
	Aggregate(other PublicKey)
	Copy() PublicKey
}

// BLSPrivateKey is a private key in Ethereum 2.
// It is a point on the BLS12-381 curve.
type BLSPrivateKey struct {
	key bls.SecretKey
}

// BLSPublicKey used in the BLS signature scheme.
type BLSPublicKey struct {
	key *bls.PublicKey
}

// BLSPrivateKeyFromBytes creates a BLS private key from a byte slice.
func BLSPrivateKeyFromBytes(priv []byte) (*BLSPrivateKey, error) {
	if len(priv) != 32 {
		return nil, errors.New("private key must be 32 bytes")
	}
	var sec bls.SecretKey
	if err := sec.Deserialize(priv); err != nil {
		return nil, errors.Wrap(err, "invalid private key")
	}
	return &BLSPrivateKey{key: sec}, nil
}

// Marshal a secret key into a byte slice.
func (p *BLSPrivateKey) Marshal() []byte {
	return p.key.Serialize()
}

// PublicKey obtains the public key corresponding to the BLS secret key.
func (p *BLSPrivateKey) PublicKey() PublicKey {
	return &BLSPublicKey{key: p.key.GetPublicKey()}
}

// Aggregate two public keys.  This updates the value of the existing key.
func (k *BLSPublicKey) Aggregate(other PublicKey) {
	k.key.Add(other.(*BLSPublicKey).key)
}

// Marshal a BLS public key into a byte slice.
func (k *BLSPublicKey) Marshal() []byte {
	return k.key.Serialize()
}

// Copy creates a copy of the public key.
func (k *BLSPublicKey) Copy() PublicKey {
	bytes := k.Marshal()
	var newKey bls.PublicKey
	//#nosec G104
	_ = newKey.Deserialize(bytes)
	return &BLSPublicKey{key: &newKey}
}
