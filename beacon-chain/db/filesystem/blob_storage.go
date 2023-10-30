package filesystem

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type BlobStorage struct {
	baseDir string
}

func (bs *BlobStorage) SaveBlobData(sidecars []*ethpb.BlobSidecar) error {
	if len(sidecars) == 0 {
		return errors.New("no blob data to save")
	}
	for _, sidecar := range sidecars {
		blobPath := path.Join(bs.baseDir, fmt.Sprintf(
			"%d_%x_%d_%x.blob",
			sidecar.Slot,
			sidecar.BlockRoot,
			sidecar.Index,
			sidecar.KzgCommitment,
		))
		exists := blobExists(blobPath)
		if exists {
			continue // Blob already exists, move to the next one
		}
		// Create a partial file and write the blob data to it.
		partialFilePath := blobPath + ".partial"
		partialFile, err := os.Create(partialFilePath)
		if err != nil {
			return errors.Wrap(err, "failed to create partial file")
		}

		_, err = partialFile.Write(sidecar.Blob)
		if err != nil {
			closeErr := partialFile.Close()
			if closeErr != nil {
				return closeErr
			}
			return errors.Wrap(err, "failed to write to partial file")
		}
		err = partialFile.Close()
		if err != nil {
			return err
		}

		// Atomically rename the partial file to its final name.
		err = os.Rename(partialFilePath, blobPath)
		if err != nil {
			return errors.Wrap(err, "failed to rename partial file to final")
		}
	}
	return nil
}

// blobExists checks that a blob file hasn't already been created and
// returns a bool representing whether it exists or not.
func blobExists(blobPath string) bool {
	// Check if the blob file already exists.
	_, err := os.Stat(blobPath)
	if err == nil {
		// The file exists.
		return true
	} else {
		// The file does not exist or an error occurred.
		return false
	}
}
