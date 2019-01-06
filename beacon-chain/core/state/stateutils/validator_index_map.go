package stateutils

import pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// ValidatorIndexMap builds a lookup map for quickly determining the index of
// a validator by their public key.
func ValidatorIndexMap(state *pb.BeaconState) map[[32]byte]int {
	m := make(map[[32]byte]int)
	for idx, record := range state.ValidatorRegistry {
		var key [32]byte
		copy(key[:], record.Pubkey)
		m[key] = idx
	}
	return m
}

// BytesToBytes32 is a convenience method for converting a byte slice to a fix
// sized 32 byte array. This method will truncate the input if it is larger
// than 32 bytes.
func BytesToBytes32(a []byte) [32]byte {
	var b [32]byte
	copy(b[:], a)
	return b
}
