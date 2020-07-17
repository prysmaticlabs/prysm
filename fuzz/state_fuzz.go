package fuzz

import pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// BeaconStateFuzz --
func BeaconStateFuzz(input []byte) int {
	st := &pb.BeaconState{}
	if err := st.UnmarshalSSZ(input); err != nil {
		return -1
	}
	return 1
}