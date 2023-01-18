package interop

import (
	"fmt"
	"os"
	"path"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// WriteBlockToDisk as a block ssz. Writes to temp directory. Debug!
func WriteBlockToDisk(prefix string, block interfaces.SignedBeaconBlock, failed bool) {
	if !features.Get().WriteSSZStateTransitions {
		return
	}

	filename := fmt.Sprintf(prefix+"_beacon_block_%d.ssz", block.Block().Slot())
	if failed {
		filename = "failed_" + filename
	}
	fp := path.Join(os.TempDir(), filename)
	log.Warnf("Writing block to disk at %s", fp)
	enc, err := block.MarshalSSZ()
	if err != nil {
		log.WithError(err).Error("Failed to ssz encode block")
		return
	}
	if err := file.WriteFile(fp, enc); err != nil {
		log.WithError(err).Error("Failed to write to disk")
	}
}

func WriteBadBlobsToDisk(prefix string, sideCar *eth.BlobsSidecar) {
	if !features.Get().WriteSSZStateTransitions {
		return
	}

	filename := fmt.Sprintf(prefix+"_blobs_%d.ssz", sideCar.BeaconBlockSlot)
	fp := path.Join(os.TempDir(), filename)
	log.Warnf("Writing blobs to disk at %s", fp)
	enc, err := sideCar.MarshalSSZ()
	if err != nil {
		log.WithError(err).Error("Failed to ssz encode blobs")
		return
	}
	if err := file.WriteFile(fp, enc); err != nil {
		log.WithError(err).Error("Failed to write to disk")
	}
}
