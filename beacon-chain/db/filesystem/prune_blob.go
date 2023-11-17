package filesystem

import (
	"context"
	"encoding/binary"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

const bufferEpochs = 2

// PruneBlobWithDB prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpoch+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) PruneBlobWithDB(ctx context.Context, currentSlot primitives.Slot, s *kv.Store) error {
	folders, err := os.ReadDir(bs.baseDir)
	if err != nil {
		return err
	}
	blockList := make(map[[32]byte]interface{})
	var pruneSlot primitives.Slot
	retentionSlot, err := slots.EpochStart(bs.retentionEpoch + bufferEpochs)
	if err != nil {
		return err
	}
	if currentSlot < retentionSlot {
		return nil // Overflow would occur
	}
	pruneSlot = currentSlot - retentionSlot
	filter := filters.NewFilter().SetStartSlot(bs.lastPrunedSlot).SetEndSlot(pruneSlot).SetBlockRoots(blockList)
	dbRoots, err := s.BlockRoots(ctx, filter)
	if err != nil {
		return err
	}
	if len(dbRoots) == 0 {
		return nil
	}

	for _, folder := range folders {
		if folder.IsDir() {
			root, err := hexutil.Decode(folder.Name())
			if err != nil {
				return err
			}
			blockList[bytesutil.ToBytes32(root)] = nil
		}
	}

	for _, root := range dbRoots {
		if _, exists := blockList[root]; exists {
			if err = os.RemoveAll(path.Join(bs.baseDir, hexutil.Encode(root[:]))); err != nil {
				return err
			}
		}
	}
	bs.lastPrunedSlot = pruneSlot
	return nil
}

// PruneBlobViaSlotFile prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpoch+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) PruneBlobViaSlotFile(currentSlot primitives.Slot) error {
	folders, err := os.ReadDir(bs.baseDir)
	if err != nil {
		return err
	}
	var slot uint64
	retentionSlot, err := slots.EpochStart(bs.retentionEpoch + bufferEpochs)
	if err != nil {
		return err
	}
	if currentSlot < retentionSlot {
		return nil // Overflow would occur
	}

	for _, folder := range folders {
		if folder.IsDir() {
			files, err := os.ReadDir(path.Join(bs.baseDir, folder.Name()))
			if err != nil {
				return err
			}
			for _, file := range files {
				ok := strings.Contains(file.Name(), ".slot")
				if !ok {
					continue
				}

				rawSlot := strings.Trim(file.Name(), ".slot")
				slot, err = strconv.ParseUint(rawSlot, 10, 64)
				if err != nil {
					return err
				}
				if primitives.Slot(slot) < (currentSlot - retentionSlot) {
					if err = os.RemoveAll(path.Join(bs.baseDir, folder.Name())); err != nil {
						return errors.Wrapf(err, "failed to delete blob %s", file.Name())
					}
				}
			}
		}
	}
	return nil
}

// PruneBlobViaRead prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpoch+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) PruneBlobViaRead(currentSlot primitives.Slot) error {
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
			files, err := os.ReadDir(path.Join(bs.baseDir, folder.Name()))
			if err != nil {
				return err
			}
			file := files[0]
			if strings.Contains(file.Name(), ".slot") {
				file = files[1]
			}
			f, err := os.Open(path.Join(bs.baseDir, folder.Name(), file.Name()))
			if err != nil {
				return err
			}
			defer f.Close()

			slot, err := slotFromBlob(f)
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
