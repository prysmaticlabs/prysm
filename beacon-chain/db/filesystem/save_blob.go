package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
)

type BlobStorage struct {
	baseDir string
}

// SaveBlobData saves blobs given a list of sidecars.
func (bs *BlobStorage) SaveBlobData(sidecars []*eth.BlobSidecar) error {
	if len(sidecars) == 0 {
		return errors.New("no blob data to save")
	}
	for _, sidecar := range sidecars {
		blobPath := bs.sidecarFileKey(sidecar)
		exists := file.Exists(blobPath)
		if exists {
			if err := checkDataIntegrity(sidecar, blobPath); err != nil {
				// This error should never happen, if it does then the
				// file has most likely been tampered with.
				return errors.Wrapf(err, "failed to save blob sidecar, tried to overwrite blob (%s) with different content", blobPath)
			}
			continue // Blob already exists, move to the next one
		}

		// Serialize the ethpb.BlobSidecar to binary data using SSZ.
		sidecarData, err := ssz.MarshalSSZ(sidecar)
		if err != nil {
			return errors.Wrap(err, "failed to serialize sidecar data")
		}

		// Create a partial file and write the serialized data to it.
		partialFilePath := blobPath + ".partial"
		partialFile, err := os.Create(filepath.Clean(partialFilePath))
		if err != nil {
			return errors.Wrap(err, "failed to create partial file")
		}

		_, err = partialFile.Write(sidecarData)
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
			return errors.Wrap(err, "failed to rename partial file to final name")
		}
	}
	return nil
}

func (bs *BlobStorage) sidecarFileKey(sidecar *eth.BlobSidecar) string {
	return path.Join(bs.baseDir, fmt.Sprintf(
		"%d_%x_%d_%x.blob",
		sidecar.Slot,
		sidecar.BlockRoot,
		sidecar.Index,
		sidecar.KzgCommitment,
	))
}

// checkDataIntegrity checks the data integrity by comparing the original ethpb.BlobSidecar.
func checkDataIntegrity(sidecar *eth.BlobSidecar, filePath string) error {
	sidecarData, err := ssz.MarshalSSZ(sidecar)
	if err != nil {
		return errors.Wrap(err, "failed to serialize sidecar data")
	}
	originalChecksum := sha256.Sum256(sidecarData)
	savedFileChecksum, err := file.HashFile(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to calculate saved file checksum")
	}
	if hex.EncodeToString(originalChecksum[:]) != hex.EncodeToString(savedFileChecksum) {
		return errors.New("data integrity check failed")
	}
	return nil
}
