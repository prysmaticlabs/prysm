package attestations

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations/kv"
)

var _ = AttestationPool(&kv.AttCaches{})
