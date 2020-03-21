package keymanager

import (
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Direct is a key manager that holds all secret keys directly.
type Direct struct {
	// Key to the map is the bytes of the public key.
	publicKeys map[params.KeyBytes]*bls.PublicKey
	// Key to the map is the bytes of the public key.
	secretKeys map[params.KeyBytes]*bls.SecretKey
}

// NewDirect creates a new direct key manager from the secret keys provided to it.
func NewDirect(sks []*bls.SecretKey) *Direct {
	res := &Direct{
		publicKeys: make(map[params.KeyBytes]*bls.PublicKey),
		secretKeys: make(map[params.KeyBytes]*bls.SecretKey),
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
func (km *Direct) FetchValidatingKeys() ([]params.KeyBytes, error) {
	keys := make([]params.KeyBytes, 0, len(km.publicKeys))
	for key := range km.publicKeys {
		keys = append(keys, key)
	}
	return keys, nil
}

// Sign signs a message for the validator to broadcast.
func (km *Direct) Sign(pubKey params.KeyBytes, root params.RootBytes, domain uint64) (*bls.Signature, error) {
	if secretKey, exists := km.secretKeys[pubKey]; exists {
		return secretKey.Sign(root[:], domain), nil
	}
	return nil, ErrNoSuchKey
}
