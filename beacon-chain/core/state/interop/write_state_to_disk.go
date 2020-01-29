package interop

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// WriteStateToDisk as a state ssz. Writes to temp directory. Debug!
func WriteStateToDisk(state *stateTrie.BeaconState) {
	if !featureconfig.Get().WriteSSZStateTransitions {
		return
	}
	fp := path.Join(os.TempDir(), fmt.Sprintf("beacon_state_%d.ssz", state.Slot()))
	log.Warnf("Writing state to disk at %s", fp)
	enc, err := ssz.Marshal(state.InnerStateUnsafe())
	if err != nil {
		log.WithError(err).Error("Failed to ssz encode state")
		return
	}
	if err := ioutil.WriteFile(fp, enc, 0664); err != nil {
		log.WithError(err).Error("Failed to write to disk")
	}
}
