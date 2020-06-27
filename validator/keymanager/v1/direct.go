package v1

import (
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// Direct is a key manager that holds all secret keys directly.
type Direct struct {
	// Key to the map is the bytes of the public key.
	publicKeys map[[48]byte]bls.PublicKey
	// Key to the map is the bytes of the public key.
	secretKeys map[[48]byte]bls.SecretKey
}

// NewDirect creates a new direct key manager from the secret keys provided to it.
func NewDirect(sks []bls.SecretKey) *Direct {
	res := &Direct{
		publicKeys: make(map[[48]byte]bls.PublicKey),
		secretKeys: make(map[[48]byte]bls.SecretKey),
	}
	for _, sk := range sks {
		publicKey := sk.PublicKey()
		pubKey := bytesutil.ToBytes48(publicKey.Marshal())
		res.publicKeys[pubKey] = publicKey
		res.secretKeys[pubKey] = sk
	}
	return res
}

// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
func (km *Direct) FetchValidatingKeys() ([][48]byte, error) {
	keys := make([][48]byte, 0, len(km.publicKeys))
	for key := range km.publicKeys {
		keys = append(keys, key)
	}
	return keys, nil
}

// Sign signs a message for the validator to broadcast.
func (km *Direct) Sign(pubKey [48]byte, root [32]byte) (bls.Signature, error) {
	if secretKey, exists := km.secretKeys[pubKey]; exists {
		return secretKey.Sign(root[:]), nil
	}
	return nil, ErrNoSuchKey
}
