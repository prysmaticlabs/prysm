package testing

import (

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"strconv"
)

const fileBase = "0-11-0/mainnet/beaconstate"

// GetBeaconFuzzState returns a beacon state by ID using the beacon-fuzz corpora.
func GetBeaconFuzzState(ID uint16) (*pb.BeaconState, error) {
	b, err := testutil.BazelFileBytes(fileBase, strconv.Itoa(int(ID)))
	if err != nil {
		return nil, err
	}
	st := &pb.BeaconState{}
	if err := st.UnmarshalSSZ(b); err != nil {
		return nil, err
	}
	return st, nil
}
