package stateutils

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ValidatorIndexMap builds a lookup map for quickly determining the index of
// a validator by their public key.
func ValidatorIndexMap(state *pb.BeaconState) map[[32]byte]int {
	m := make(map[[32]byte]int)
	for idx, record := range state.ValidatorRegistry {
		key := bytesutil.ToBytes32(record.Pubkey)
		m[key] = idx
	}
	return m
}
