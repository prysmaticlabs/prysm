package stateutils

import (
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ValidatorIndexMap builds a lookup map for quickly determining the index of
// a validator by their public key.
func ValidatorIndexMap(state *stateTrie.BeaconState) map[[48]byte]int {
	m := make(map[[48]byte]int)
	vals := state.Validators()
	for idx, record := range vals {
		key := bytesutil.ToBytes48(record.PublicKey)
		m[key] = idx
	}
	return m
}
