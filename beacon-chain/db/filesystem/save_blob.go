package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type BlobStorage struct {
	baseDir string
}

// SaveBlobData saves blobs given a list of sidecars.
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
		exists := file.Exists(blobPath)
		if exists {
			if err := checkDataIntegrity(sidecar.Blob, blobPath); err != nil {
				return errors.Wrapf(err, "failed to save blob sidecar, tried to overwrite blob (%s) with different content", blobPath)
			}
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

		if err = checkDataIntegrity(sidecar.Blob, blobPath); err != nil {
			return err
		}
	}
	return nil
}

// checkDataIntegrity checks the data integrity by comparing SHA256 checksums.
func checkDataIntegrity(originalData []byte, filePath string) error {
	originalChecksum := sha256.Sum256(originalData)
	savedFileChecksum, err := file.HashFile(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to calculate saved file checksum")
	}
	if hex.EncodeToString(originalChecksum[:]) != hex.EncodeToString(savedFileChecksum) {
		return errors.New("data integrity check failed")
	}
	return nil
}
