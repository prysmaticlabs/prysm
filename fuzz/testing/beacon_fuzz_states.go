package testing

import (
	"fmt"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"os"
	"strconv"
)

const fileBase = "0-11-0/mainnet/beaconstate"
const fileBaseENV = "BEACONSTATES_PATH"

// GetBeaconFuzzState returns a beacon state by ID using the beacon-fuzz corpora.
func GetBeaconFuzzStateBytes(ID uint16) ([]byte, error) {
	base := fileBase
	// Using an environment variable allows a host image to specify the path when only the binary
	// executable was uploaded (without the runfiles). i.e. fuzzit's platform.
	if p, ok := os.LookupEnv(fileBaseENV); ok {
		base = p
	}
	ok, err := testutil.BazelDirectoryNonEmpty(base)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic(fmt.Sprintf("Beacon states directory (%s) does not exist or has no files.", base))
	}
	return testutil.BazelFileBytes(base, strconv.Itoa(int(ID)))
}

// GetBeaconFuzzState returns a beacon state by ID using the beacon-fuzz corpora.
// Deprecated: Prefer GetBeaconFuzzStateBytes(ID) and handle ssz marshal in the caller method.
func GetBeaconFuzzState(ID uint16) (*pb.BeaconState, error) {
	b, err := GetBeaconFuzzStateBytes(ID)
	if err != nil {
		return nil, err
	}
	st := &pb.BeaconState{}
	if err := st.UnmarshalSSZ(b); err != nil {
		return nil, err
	}
	return st, nil
}
