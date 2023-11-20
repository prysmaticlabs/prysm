package filesystem

import (
	"encoding/binary"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

const bufferEpochs = 2

// Prune prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpoch+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) Prune(currentSlot primitives.Slot) error {
	folders, err := os.ReadDir(bs.baseDir)
	if err != nil {
		return err
	}
	retentionSlot, err := slots.EpochStart(bs.retentionEpoch + bufferEpochs)
	if err != nil {
		return err
	}
	if currentSlot < retentionSlot {
		return nil // Overflow would occur
	}

	for _, folder := range folders {
		if folder.IsDir() {
			folderPath := path.Join(bs.baseDir, folder.Name())
			file, err := os.Open(folderPath + "/0.blob")
			if err != nil {
				return err
			}
			defer file.Close()

			slot, err := slotFromBlob(file)
			if err != nil {
				return err
			}
			if slot < (currentSlot - retentionSlot) {
				if err = os.RemoveAll(path.Join(bs.baseDir, folder.Name())); err != nil {
					return errors.Wrapf(err, "failed to delete blob %s", file.Name())
				}
			}
		}
	}
	return nil
}

func slotFromBlob(at io.ReaderAt) (primitives.Slot, error) {
	b := make([]byte, 8)
	_, err := at.ReadAt(b, 40)
	if err != nil {
		return 0, err
	}
	rawSlot := binary.LittleEndian.Uint64(b)
	return primitives.Slot(rawSlot), nil
}
