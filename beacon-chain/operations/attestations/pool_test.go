package attestations

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations/kv"
)

var _ Pool = (*kv.AttCaches)(nil)
