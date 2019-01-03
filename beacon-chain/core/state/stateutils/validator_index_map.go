package stateutils

import pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

func ValidatorIndexMap(state *pb.BeaconState) map[[32]byte]int {
	m := make(map[[32]byte]int)
	for idx, record := range state.ValidatorRegistry {
		var key [32]byte
		copy(key[:], record.Pubkey)
		m[key] = idx
	}
	return m
}

func BytesToBytes32(a []byte) [32]byte {
	var b [32]byte
	copy(b[:], a)
	return b
}
