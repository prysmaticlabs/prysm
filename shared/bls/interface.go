package bls

import (
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
)

// PublicKey represents a BLS public key.
type PublicKey = iface.PublicKey

// SecretKey represents a BLS secret or private key.
type SecretKey = iface.SecretKey

// Signature represents a BLS signature.
type Signature = iface.Signature
