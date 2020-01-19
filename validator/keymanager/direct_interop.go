package keymanager

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

// Interop is a key manager that deterministically generates keys.
type Interop struct {
	*Direct
}

type interopOpts struct {
	Keys   uint64 `json:"keys"`
	Offset uint64 `json:"offset"`
}

// NewInterop creates a key manager using a number of interop keys at a given offset.
func NewInterop(input string) (*Interop, error) {
	opts := &interopOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse options")
	}

	sks, pks, err := interop.DeterministicallyGenerateKeys(opts.Offset, opts.Keys)
	if err != nil {
		return nil, err
	}

	km := &Interop{
		Direct: &Direct{
			publicKeys: make(map[[48]byte]*bls.PublicKey),
			secretKeys: make(map[[48]byte]*bls.SecretKey),
		},
	}
	for i := 0; uint64(i) < opts.Keys; i++ {
		pubKey := bytesutil.ToBytes48(pks[i].Marshal())
		km.publicKeys[pubKey] = pks[i]
		km.secretKeys[pubKey] = sks[i]
	}
	return km, nil
}
