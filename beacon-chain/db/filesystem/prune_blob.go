package filesystem

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

const bufferEpochs = 2

// PruneBlob prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpoch+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) PruneBlob(currentSlot primitives.Slot) error {
	files, err := os.ReadDir(bs.baseDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		slot, err := extractSlotFromFileName(file.Name())
		if err != nil {
			return err
		}
		retentionSlot, err := slots.EpochStart(bs.retentionEpoch + bufferEpochs)
		if err != nil {
			return err
		}
		if currentSlot < retentionSlot {
			continue // Overflow would occur
		}
		if slot < (currentSlot - retentionSlot) {
			if err = os.Remove(path.Join(bs.baseDir, file.Name())); err != nil {
				return errors.Wrapf(err, "failed to delete blob %s", file.Name())
			}
		}
	}
	return nil
}

// extractSlotFromFileName returns the slot of a blob from a given filename.
func extractSlotFromFileName(fileName string) (primitives.Slot, error) {
	parts := strings.Split(strings.TrimSuffix(fileName, ".blob"), "_")
	if len(parts) < 1 {
		return 0, errors.New("invalid file format")
	}
	slot, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse slot from filename: %s", fileName)
	}
	return primitives.Slot(slot), nil
}
