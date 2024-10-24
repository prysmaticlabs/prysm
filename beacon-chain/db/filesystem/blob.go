package filesystem

import (
	"fmt"
	"math"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	"github.com/prysmaticlabs/prysm/v5/runtime/logging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

func directoryPermissions() os.FileMode {
	return params.BeaconIoConfig().ReadWriteExecutePermissions
}

var (
	errIndexOutOfBounds    = errors.New("blob index in file name >= MaxBlobsPerBlock")
	errSidecarEmptySSZData = errors.New("sidecar marshalled to an empty ssz byte slice")
	errNoBasePath          = errors.New("BlobStorage base path not specified in init")
)

// BlobStorageOption is a functional option for configuring a BlobStorage.
type BlobStorageOption func(*BlobStorage) error

// WithBasePath is a required option that sets the base path of blob storage.
func WithBasePath(base string) BlobStorageOption {
	return func(b *BlobStorage) error {
		b.base = base
		return nil
	}
}

// WithBlobRetentionEpochs is an option that changes the number of epochs blobs will be persisted.
func WithBlobRetentionEpochs(e primitives.Epoch) BlobStorageOption {
	return func(b *BlobStorage) error {
		b.retentionEpochs = e
		return nil
	}
}

// WithSaveFsync is an option that causes Save to call fsync before renaming part files for improved durability.
func WithSaveFsync(fsync bool) BlobStorageOption {
	return func(b *BlobStorage) error {
		b.fsync = fsync
		return nil
	}
}

// WithFs allows the afero.Fs implementation to be customized. Used by tests
// to substitute an in-memory filesystem.
func WithFs(fs afero.Fs) BlobStorageOption {
	return func(b *BlobStorage) error {
		b.fs = fs
		return nil
	}
}

// WithLayout enables the user to specify a runtimeLayout which dictates how blob files are stored on disk.
func WithLayout(name string) BlobStorageOption {
	return func(b *BlobStorage) error {
		b.layoutName = name
		return nil
	}
}

// NewBlobStorage creates a new instance of the BlobStorage object. Note that the implementation of BlobStorage may
// attempt to hold a file lock to guarantee exclusive control of the blob storage directory, so this should only be
// initialized once per beacon node.
func NewBlobStorage(opts ...BlobStorageOption) (*BlobStorage, error) {
	b := &BlobStorage{}
	for _, o := range opts {
		if err := o(b); err != nil {
			return nil, errors.Wrap(err, "failed to create blob storage")
		}
	}
	// Allow tests to set up a different fs using WithFs.
	if b.fs == nil {
		if b.base == "" {
			return nil, errNoBasePath
		}
		b.base = path.Clean(b.base)
		if err := file.MkdirAll(b.base); err != nil {
			return nil, errors.Wrapf(err, "failed to create blob storage at %s", b.base)
		}
		b.fs = afero.NewBasePathFs(afero.NewOsFs(), b.base)
	}
	b.cache = newBlobStorageCache()
	pruner := newBlobPruner(b.retentionEpochs)
	if b.layoutName == "" {
		b.layoutName = LayoutNameFlat
	}
	layout, err := newLayout(b.layoutName, b.fs, b.cache, pruner)
	if err != nil {
		return nil, err
	}
	b.layout = layout
	return b, nil
}

// BlobStorage is the concrete implementation of the filesystem backend for saving and retrieving BlobSidecars.
type BlobStorage struct {
	base            string
	retentionEpochs primitives.Epoch
	layoutName      string
	fsync           bool
	fs              afero.Fs
	layout          fsLayout
	cache           *blobStorageCache
}

// WarmCache runs the prune routine with an expiration of slot of 0, so nothing will be pruned, but the pruner's cache
// will be populated at node startup, avoiding a costly cold prune (~4s in syscalls) during syncing.
func (bs *BlobStorage) WarmCache() {
	start := time.Now()
	if err := warmCache(bs.layout, bs.cache); err != nil {
		log.WithError(err).Error("Error encountered while warming up blob filesystem cache.")
	}
	if err := bs.migrateLayouts(); err != nil {
		log.WithError(err).Error("Error encountered while migrating blob storage.")
	}
	log.WithField("elapsed", time.Since(start)).Info("Blob filesystem cache warm-up complete.")
}

// If any blob storage directories are found for layouts besides the configured layout, migrate them.
func (bs *BlobStorage) migrateLayouts() error {
	for _, name := range LayoutNames {
		if name == bs.layoutName {
			continue
		}
		from, err := newLayout(name, bs.fs, bs.cache, nil)
		if err != nil {
			return err
		}
		if err := migrateLayout(bs.fs, from, bs.layout, bs.cache); err != nil {
			return errors.Wrapf(err, "failed to migrate layout from %s to %s", name, bs.layoutName)
		}
	}
	return nil
}

func (bs *BlobStorage) writePart(sidecar blocks.VerifiedROBlob) (ppath string, err error) {
	ident := identForSidecar(sidecar)
	sidecarData, err := sidecar.MarshalSSZ()
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize sidecar data")
	}
	if len(sidecarData) == 0 {
		return "", errSidecarEmptySSZData
	}

	if err := bs.fs.MkdirAll(bs.layout.dir(ident), directoryPermissions()); err != nil {
		return "", err
	}
	ppath = bs.layout.partPath(ident, fmt.Sprintf("%p", sidecarData))

	// Create a partial file and write the serialized data to it.
	partialFile, err := bs.fs.Create(ppath)
	if err != nil {
		return "", errors.Wrap(err, "failed to create partial file")
	}
	defer func() {
		cerr := partialFile.Close()
		// The close error is probably less important than any existing error, so only overwrite nil err.
		if cerr != nil && err == nil {
			err = cerr
		}
	}()

	n, err := partialFile.Write(sidecarData)
	if err != nil {
		return ppath, errors.Wrap(err, "failed to write to partial file")
	}
	if bs.fsync {
		if err := partialFile.Sync(); err != nil {
			return ppath, err
		}
	}

	if n != len(sidecarData) {
		return ppath, fmt.Errorf("failed to write the full bytes of sidecarData, wrote only %d of %d bytes", n, len(sidecarData))
	}

	return ppath, nil
}

// Save saves blobs given a list of sidecars.
func (bs *BlobStorage) Save(sidecar blocks.VerifiedROBlob) error {
	startTime := time.Now()

	ident := identForSidecar(sidecar)
	sszPath := bs.layout.sszPath(ident)
	exists, err := afero.Exists(bs.fs, sszPath)
	if err != nil {
		return err
	}
	if exists {
		log.WithFields(logging.BlobFields(sidecar.ROBlob)).Debug("Ignoring a duplicate blob sidecar save attempt")
		return nil
	}

	partialMoved := false
	partPath, err := bs.writePart(sidecar)
	// Ensure the partial file is deleted.
	defer func() {
		if partialMoved || partPath == "" {
			return
		}
		// It's expected to error if the save is successful.
		err := bs.fs.Remove(partPath)
		if err == nil {
			log.WithFields(logrus.Fields{
				"partPath": partPath,
			}).Debugf("Removed partial file")
		}
	}()
	if err != nil {
		return err
	}

	// Atomically rename the partial file to its final name.
	err = bs.fs.Rename(partPath, sszPath)
	if err != nil {
		return errors.Wrap(err, "failed to rename partial file to final name")
	}
	partialMoved = true

	if err := bs.layout.notify(ident); err != nil {
		return errors.Wrapf(err, "problem maintaining pruning cache/metrics for sidecar with root=%#x", sidecar.BlockRoot())
	}
	blobsWrittenCounter.Inc()
	blobSaveLatency.Observe(float64(time.Since(startTime).Milliseconds()))
	return nil
}

// Get retrieves a single BlobSidecar by its root and index.
// Since BlobStorage only writes blobs that have undergone full verification, the return
// value is always a VerifiedROBlob.
func (bs *BlobStorage) Get(root [32]byte, idx uint64) (blocks.VerifiedROBlob, error) {
	startTime := time.Now()
	ident, err := bs.layout.ident(root, idx)
	if err != nil {
		return verification.VerifiedROBlobError(err)
	}
	defer func() {
		blobFetchLatency.Observe(float64(time.Since(startTime).Milliseconds()))
	}()
	return verification.VerifiedROBlobFromDisk(bs.fs, root, bs.layout.sszPath(ident))
}

// Remove removes all blobs for a given root.
func (bs *BlobStorage) Remove(root [32]byte) error {
	dirIdent, err := bs.layout.dirIdent(root)
	if err != nil {
		return err
	}
	_, err = bs.layout.remove(dirIdent)
	return err
}

// Indices generates a bitmap representing which BlobSidecar.Index values are present on disk for a given root.
// This value can be compared to the commitments observed in a block to determine which indices need to be found
// on the network to confirm data availability.
func (bs *BlobStorage) Indices(root [32]byte) ([fieldparams.MaxBlobsPerBlock]bool, error) {
	return bs.Summary(root).mask, nil
}

// Summary returns the BlobStorageSummary from the layout.
// Internally, this is a cached representation of the directory listing for the given root.
func (bs *BlobStorage) Summary(root [32]byte) BlobStorageSummary {
	return bs.layout.summary(root)
}

// Clear deletes all files on the filesystem.
func (bs *BlobStorage) Clear() error {
	dirs, err := listDir(bs.fs, ".")
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := bs.fs.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

// WithinRetentionPeriod checks if the requested epoch is within the blob retention period.
func (bs *BlobStorage) WithinRetentionPeriod(requested, current primitives.Epoch) bool {
	if requested > math.MaxUint64-bs.retentionEpochs {
		// If there is an overflow, then the retention period was set to an extremely large number.
		return true
	}
	return requested+bs.retentionEpochs >= current
}
