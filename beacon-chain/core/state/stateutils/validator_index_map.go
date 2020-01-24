package stateutils

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ValidatorIndexMap builds a lookup map for quickly determining the index of
// a validator by their public key.
func ValidatorIndexMap(state *pb.BeaconState) map[[48]byte]uint64 {
	m := make(map[[48]byte]uint64)
	vals := state.Validators
	for idx, record := range vals {
		key := bytesutil.ToBytes48(record.PublicKey)
		m[key] = uint64(idx)
	}
	return m
}
