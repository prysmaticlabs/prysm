package keymanager

import (
	"encoding/json"

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

var interopOptsHelp = `The interop key manager generates keys according to the interop testnet specification.  The options are:
  - keys This is the number of keys to generate
  - offset This is the number of keys to skip before starting to generate keys
A sample set of options are:
  {
    "keys":   5,   // Generate 5 keys
    "offset": 1024 // Start with the 1024th key
  }`

// NewInterop creates a key manager using a number of interop keys at a given offset.
func NewInterop(input string) (*Interop, string, error) {
	opts := &interopOpts{}
	err := json.Unmarshal([]byte(input), opts)
	if err != nil {
		return nil, interopOptsHelp, err
	}

	sks, pks, err := interop.DeterministicallyGenerateKeys(opts.Offset, opts.Keys)
	if err != nil {
		return nil, interopOptsHelp, err
	}

	km := &Interop{
		Direct: &Direct{
			publicKeys: make(map[[48]byte]bls.PublicKey),
			secretKeys: make(map[[48]byte]bls.SecretKey),
		},
	}
	for i := 0; uint64(i) < opts.Keys; i++ {
		pubKey := bytesutil.ToBytes48(pks[i].Marshal())
		km.publicKeys[pubKey] = pks[i]
		km.secretKeys[pubKey] = sks[i]
	}
	return km, "", nil
}
