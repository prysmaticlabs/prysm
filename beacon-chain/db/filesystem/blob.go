package filesystem

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/io/file"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/logging"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var (
	errIndexOutOfBounds = errors.New("blob index in file name >= MaxBlobsPerBlock")
)

const (
	sszExt  = "ssz"
	partExt = "part"

	firstPruneEpoch      = 0
	bufferEpochs         = 2
	directoryPermissions = 0700
)

// BlobStorageOption is a functional option for configuring a BlobStorage.
type BlobStorageOption func(*BlobStorage)

// WithBlobRetentionEpochs is an option that changes the number of epochs blobs will be persisted.
func WithBlobRetentionEpochs(e primitives.Epoch) BlobStorageOption {
	return func(b *BlobStorage) {
		b.retentionEpochs = e
	}
}

// NewBlobStorage creates a new instance of the BlobStorage object. Note that the implementation of BlobStorage may
// attempt to hold a file lock to guarantee exclusive control of the blob storage directory, so this should only be
// initialized once per beacon node.
func NewBlobStorage(base string, opts ...BlobStorageOption) (*BlobStorage, error) {
	base = path.Clean(base)
	if err := file.MkdirAll(base); err != nil {
		return nil, fmt.Errorf("failed to create blob storage at %s: %w", base, err)
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), base)
	b := &BlobStorage{
		fs:              fs,
		retentionEpochs: params.BeaconConfig().MinEpochsForBlobsSidecarsRequest,
		lastPrunedEpoch: firstPruneEpoch,
	}
	for _, o := range opts {
		o(b)
	}
	return b, nil
}

// BlobStorage is the concrete implementation of the filesystem backend for saving and retrieving BlobSidecars.
type BlobStorage struct {
	fs              afero.Fs
	retentionEpochs primitives.Epoch
	lastPrunedEpoch primitives.Epoch
}

// Save saves blobs given a list of sidecars.
func (bs *BlobStorage) Save(sidecar blocks.VerifiedROBlob) error {
	startTime := time.Now()
	fname := namerForSidecar(sidecar)
	sszPath := fname.path()
	exists, err := afero.Exists(bs.fs, sszPath)
	if err != nil {
		return err
	}
	if exists {
		log.WithFields(logging.BlobFields(sidecar.ROBlob)).Debug("ignoring a duplicate blob sidecar Save attempt")
		return nil
	}
	if bs.shouldPrune(sidecar.Slot()) {
		go func() {
			err := bs.pruneOlderThan(sidecar.Slot())
			if err != nil {
				log.WithError(err).Errorf("failed to prune blobs from slot %d", sidecar.Slot())
			}
		}()
	}

	// Serialize the ethpb.BlobSidecar to binary data using SSZ.
	sidecarData, err := sidecar.MarshalSSZ()
	if err != nil {
		return errors.Wrap(err, "failed to serialize sidecar data")
	}
	if err := bs.fs.MkdirAll(fname.dir(), directoryPermissions); err != nil {
		return err
	}
	partPath := fname.partPath()

	// Ensure the partial file is deleted.
	defer func() {
		// It's expected to error if the save is successful.
		err = bs.fs.Remove(partPath)
		if err == nil {
			log.WithFields(log.Fields{
				"partPath": partPath,
			}).Debugf("removed partial file")
		}
	}()

	// Create a partial file and write the serialized data to it.
	partialFile, err := bs.fs.Create(partPath)
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
	err = bs.fs.Rename(partPath, sszPath)
	if err != nil {
		return errors.Wrap(err, "failed to rename partial file to final name")
	}
	blobsTotalGauge.Inc()
	blobSaveLatency.Observe(time.Since(startTime).Seconds())
	return nil
}

// Get retrieves a single BlobSidecar by its root and index.
// Since BlobStorage only writes blobs that have undergone full verification, the return
// value is always a VerifiedROBlob.
func (bs *BlobStorage) Get(root [32]byte, idx uint64) (blocks.VerifiedROBlob, error) {
	startTime := time.Now()
	expected := blobNamer{root: root, index: idx}
	encoded, err := afero.ReadFile(bs.fs, expected.path())
	var v blocks.VerifiedROBlob
	if err != nil {
		return v, err
	}
	s := &ethpb.BlobSidecar{}
	if err := s.UnmarshalSSZ(encoded); err != nil {
		return v, err
	}
	ro, err := blocks.NewROBlobWithRoot(s, root)
	if err != nil {
		return blocks.VerifiedROBlob{}, err
	}
	defer func() {
		blobFetchLatency.Observe(time.Since(startTime).Seconds())
	}()
	return verification.BlobSidecarNoop(ro)
}

// Indices generates a bitmap representing which BlobSidecar.Index values are present on disk for a given root.
// This value can be compared to the commitments observed in a block to determine which indices need to be found
// on the network to confirm data availability.
func (bs *BlobStorage) Indices(root [32]byte) ([fieldparams.MaxBlobsPerBlock]bool, error) {
	var mask [fieldparams.MaxBlobsPerBlock]bool
	rootDir := blobNamer{root: root}.dir()
	entries, err := afero.ReadDir(bs.fs, rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return mask, nil
		}
		return mask, err
	}
	for i := range entries {
		if entries[i].IsDir() {
			continue
		}
		name := entries[i].Name()
		if !strings.HasSuffix(name, sszExt) {
			continue
		}
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			continue
		}
		u, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return mask, errors.Wrapf(err, "unexpected directory entry breaks listing, %s", parts[0])
		}
		if u >= fieldparams.MaxBlobsPerBlock {
			return mask, errIndexOutOfBounds
		}
		mask[u] = true
	}
	return mask, nil
}

type blobNamer struct {
	root  [32]byte
	index uint64
}

func namerForSidecar(sc blocks.VerifiedROBlob) blobNamer {
	return blobNamer{root: sc.BlockRoot(), index: sc.Index}
}

func (p blobNamer) dir() string {
	return fmt.Sprintf("%#x", p.root)
}

func (p blobNamer) fname(ext string) string {
	return path.Join(p.dir(), fmt.Sprintf("%d.%s", p.index, ext))
}

func (p blobNamer) partPath() string {
	return p.fname(partExt)
}

func (p blobNamer) path() string {
	return p.fname(sszExt)
}

// Prune prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpochs+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (bs *BlobStorage) Prune(currentSlot primitives.Slot) error {
	t := time.Now()
	retentionSlots, err := slots.EpochStart(bs.retentionEpochs + bufferEpochs)
	if err != nil {
		return err
	}
	if currentSlot < retentionSlots {
		return nil // Overflow would occur
	}

	log.Debug("Pruning old blobs")

	folders, err := afero.ReadDir(bs.fs, ".")
	if err != nil {
		return err
	}
	var totalPruned int
	for _, folder := range folders {
		if folder.IsDir() {
			num, err := bs.processFolder(folder, currentSlot, retentionSlots)
			if err != nil {
				return err
			}
			blobsPrunedCounter.Add(float64(num))
			blobsTotalGauge.Add(-float64(num))
			totalPruned += num
		}
	}
	pruneTime := time.Since(t)

	log.WithFields(log.Fields{
		"lastPrunedEpoch":   slots.ToEpoch(currentSlot - retentionSlots),
		"pruneTime":         pruneTime,
		"numberBlobsPruned": totalPruned,
	}).Debug("Pruned old blobs")

	return nil
}

// processFolder will delete the folder of blobs if the blob slot is outside the
// retention period. We determine the slot by looking at the first blob in the folder.
func (bs *BlobStorage) processFolder(folder os.FileInfo, currentSlot, retentionSlots primitives.Slot) (int, error) {
	f, err := bs.fs.Open(filepath.Join(folder.Name(), "0."+sszExt))
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.WithError(err).Errorf("Could not close blob file")
		}
	}()

	slot, err := slotFromBlob(f)
	if err != nil {
		return 0, err
	}
	var num int
	if slot < (currentSlot - retentionSlots) {
		num, err = bs.countFiles(folder.Name())
		if err != nil {
			return 0, err
		}
		if err = bs.fs.RemoveAll(folder.Name()); err != nil {
			return 0, errors.Wrapf(err, "failed to delete blob %s", f.Name())
		}
	}
	return num, nil
}

// slotFromBlob reads the ssz data of a file at the specified offset (8 + 131072 + 48 + 48 = 131176 bytes),
// which is calculated based on the size of the BlobSidecar struct and is based on the size of the fields
// preceding the slot information within SignedBeaconBlockHeader.
func slotFromBlob(at io.ReaderAt) (primitives.Slot, error) {
	b := make([]byte, 8)
	_, err := at.ReadAt(b, 131176)
	if err != nil {
		return 0, err
	}
	rawSlot := binary.LittleEndian.Uint64(b)
	return primitives.Slot(rawSlot), nil
}

// Delete removes the directory matching the provided block root and all the blobs it contains.
func (bs *BlobStorage) Delete(root [32]byte) error {
	if err := bs.fs.RemoveAll(hexutil.Encode(root[:])); err != nil {
		return fmt.Errorf("failed to delete blobs for root %#x: %w", root, err)
	}
	return nil
}

// shouldPrune checks whether pruning should be triggered based on the given slot.
func (bs *BlobStorage) shouldPrune(slot primitives.Slot) bool {
	if slots.SinceEpochStarts(slot) < params.BeaconConfig().SlotsPerEpoch/2 {
		return false
	}
	if slots.ToEpoch(slot) == bs.lastPrunedEpoch {
		return false
	}
	return true
}

// pruneOlderThan prunes blobs in the base directory based on the retention epoch and current slot.
func (bs *BlobStorage) pruneOlderThan(slot primitives.Slot) error {
	err := bs.Prune(slot)
	if err != nil {
		return err
	}
	// Update lastPrunedEpoch to the current epoch.
	bs.lastPrunedEpoch = slots.ToEpoch(slot)
	return nil
}
