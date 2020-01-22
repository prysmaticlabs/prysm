package stateutils

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ValidatorIndexMap builds a lookup map for quickly determining the index of
// a validator by their public key.
func ValidatorIndexMap(validators []*ethpb.Validator) map[[48]byte]int {
	m := make(map[[48]byte]int)
	for idx, record := range validators {
		key := bytesutil.ToBytes48(record.PublicKey)
		m[key] = idx
	}
	return m
}
