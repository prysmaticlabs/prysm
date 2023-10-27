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
		filepath, exists := blobExists(bs.baseDir, sidecar)
		if exists {
			continue // Blob already exists, move to the next one
		}
		// Create a partial file and write the blob data to it.
		partialFilePath := filepath + ".partial"
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
		err = os.Rename(partialFilePath, filepath)
		if err != nil {
			return errors.Wrap(err, "failed to rename partial file to final")
		}
	}
	return nil
}

// blobExists checks that a blob file hasn't already been created and
// returns the path and a bool representing whether it exists or not.
func blobExists(baseDir string, sidecar *ethpb.BlobSidecar) (string, bool) {
	blobPath := path.Join(baseDir, fmt.Sprintf(
		"%d_%x_%d_%x.blob",
		sidecar.Slot,
		sidecar.BlockRoot,
		sidecar.Index,
		sidecar.KzgCommitment,
	))
	// Check if the blob file already exists.
	_, err := os.Stat(blobPath)
	if err == nil {
		// The file exists.
		return blobPath, true
	} else if os.IsNotExist(err) {
		// The file does not exist.
		return blobPath, false
	} else {
		// An error occurred while checking the file.
		return blobPath, false
	}
}
