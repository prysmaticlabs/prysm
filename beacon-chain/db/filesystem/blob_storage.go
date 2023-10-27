package filesystem

import (
	"fmt"
	"os"
	"path"

	"github.com/alexflint/go-filemutex"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type BlobStorage struct {
	baseDir  string
	fileLock *filemutex.FileMutex
}

// newFileMutex creates a new file mutex for a file given its path.
func (bs *BlobStorage) newFileMutex(path string) error {
	fileLock, err := filemutex.New(path)
	if err != nil {
		return err
	}
	bs.fileLock = fileLock
	return nil
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

		err := bs.newFileMutex(partialFilePath)
		if err != nil {
			return errors.Wrap(err, "failed to create file lock")
		}
		err = bs.fileLock.Lock()
		if err != nil {
			return errors.Wrap(err, "failed to acquire file lock")
		}
		defer func() {
			if err := bs.fileLock.Unlock(); err != nil {
				log.Errorf("Error releasing mutex%v", err)
			}
		}()

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
