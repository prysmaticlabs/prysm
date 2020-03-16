package testing

import (
	"encoding/hex"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Returns a beacon state by ID. Returns nil if not found. This function depends on generated states
// from //tools/beacon-fuzz tool. These states are generated and hard coded into the application
// source and will be loaded into memory at runtime.
func GetBeaconFuzzState(ID uint16) *pb.BeaconState {
	if s, ok := generatedStates[ID]; ok {
		b, err := hex.DecodeString(s)
		if err != nil {
			panic(err)
		}
		st := &pb.BeaconState{}
		if err := st.UnmarshalSSZ(b); err != nil {
			panic(err)
		}
		return st
	}
	return nil
}
