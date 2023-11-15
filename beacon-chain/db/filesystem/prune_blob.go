package filesystem

import (
	"context"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

const bufferEpochs = 2

// PruneBlobWithDB prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpoch+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) PruneBlobWithDB(ctx context.Context, currentSlot primitives.Slot, s *kv.Store) error {
	files, err := os.ReadDir(bs.baseDir)
	if err != nil {
		return err
	}
	var blockList [][]byte
	var pruneSlot primitives.Slot
	for _, file := range files {
		root, err := hexutil.Decode("0x" + file.Name())
		if err != nil {
			return err
		}
		blockList = append(blockList, root)
	}
	for _, file := range files {
		if file.IsDir() {
			retentionSlot, err := slots.EpochStart(bs.retentionEpoch + bufferEpochs)
			if err != nil {
				return err
			}
			if currentSlot < retentionSlot {
				continue // Overflow would occur
			}

			pruneSlot = currentSlot - retentionSlot
			filter := filters.NewFilter().SetStartSlot(bs.lastPrunedSlot).SetEndSlot(pruneSlot).SetBlockRoots(blockList)
			roots, err := s.BlockRoots(ctx, filter)
			if err != nil {
				return err
			}

			fileRoot, err := hexutil.Decode(file.Name())
			if err != nil {
				return err
			}

			for _, root := range roots {
				if root == [32]byte(fileRoot) {
					if err = os.RemoveAll(path.Join(bs.baseDir, file.Name())); err != nil {
						return errors.Wrapf(err, "failed to delete blob %s", file.Name())
					}
				}
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
				retentionSlot, err := slots.EpochStart(bs.retentionEpoch + bufferEpochs)
				if err != nil {
					return err
				}
				if currentSlot < retentionSlot {
					continue // Overflow would occur
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
