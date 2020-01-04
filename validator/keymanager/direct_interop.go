package keymanager

import (
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

// Interop is a key manager that deterministically generates keys.
type Interop struct {
	*Direct
}

// NewInterop creates a key manager using the given number of interop keys at the given offset.
func NewInterop(keys uint64, offset uint64) (*Interop, error) {
	sks, pks, err := interop.DeterministicallyGenerateKeys(offset, keys)
	if err != nil {
		return nil, err
	}

	km := &Interop{
		Direct: &Direct{
			publicKeys: make(map[[48]byte]*bls.PublicKey),
			secretKeys: make(map[[48]byte]*bls.SecretKey),
		},
	}
	for i := 0; uint64(i) < keys; i++ {
		pubKey := bytesutil.ToBytes48(pks[i].Marshal())
		km.publicKeys[pubKey] = pks[i]
		km.secretKeys[pubKey] = sks[i]
	}
	return km, nil
}
