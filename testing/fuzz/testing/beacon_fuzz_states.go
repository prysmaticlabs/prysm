package testing

import (
	"fmt"
	"os"
	"strconv"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/util"
)

const fileBase = "0-11-0/mainnet/beaconstate"
const fileBaseENV = "BEACONSTATES_PATH"

// BeaconFuzzState returns a beacon state by ID using the beacon-fuzz corpora.
func BeaconFuzzState(ID uint64) (*ethpb.BeaconState, error) {
	base := fileBase
	// Using an environment variable allows a host image to specify the path when only the binary
	// executable was uploaded (without the runfiles). i.e. fuzzit's platform.
	if p, ok := os.LookupEnv(fileBaseENV); ok {
		base = p
	}
	ok, err := util.BazelDirectoryNonEmpty(base)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic(fmt.Sprintf("Beacon states directory (%s) does not exist or has no files.", base))
	}
	b, err := util.BazelFileBytes(base, strconv.Itoa(int(ID)))
	if err != nil {
		return nil, err
	}
	st := &ethpb.BeaconState{}
	if err := st.UnmarshalSSZ(b); err != nil {
		return nil, err
	}
	return st, nil
}
