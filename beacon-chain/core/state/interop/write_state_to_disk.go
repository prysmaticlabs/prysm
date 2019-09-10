package interop

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// WriteStateToDisk as a state ssz. Writes to temp directory. Debug!
func WriteStateToDisk(state *pb.BeaconState) {
	fp := path.Join(os.TempDir(), fmt.Sprintf("beacon_state_%d.ssz", state.Slot))
	log.Warnf("Writing state to disk at %s", fp)
	enc, err := ssz.Marshal(state)
	if err != nil {
		log.WithError(err).Error("Failed to ssz encode state")
		return
	}
	if err := ioutil.WriteFile(fp, enc, 0664); err != nil {
		log.WithError(err).Error("Failed to write to disk")
	}
}
