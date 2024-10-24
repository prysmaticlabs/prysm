package filesystem

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const (
	// Full root in directory will be 66 chars, eg:
	// >>> len('0x0002fb4db510b8618b04dc82d023793739c26346a8b02eb73482e24b0fec0555') == 66
	rootStringLen        = 66
	sszExt               = "ssz"
	partExt              = "part"
	periodicEpochBaseDir = "by-epoch"
)

const (
	LayoutNameFlat    = "flat"
	LayoutNameByEpoch = "by-epoch"
)

var LayoutNames = []string{LayoutNameFlat, LayoutNameByEpoch}

var (
	errMigrationFailure       = errors.New("unable to migrate blob directory between old and new layout")
	errCacheWarmFailed        = errors.New("failed to warm blob filesystem cache")
	errPruneFailed            = errors.New("failed to prune root")
	errInvalidRootString      = errors.New("Could not parse hex string as a [32]byte")
	errInvalidDirectoryLayout = errors.New("Could not parse blob directory path")
	errInvalidLayoutName      = errors.New("unknown layout name")
)

type blobIdent struct {
	root  [32]byte
	epoch primitives.Epoch
	index uint64
}

func newBlobIdent(root [32]byte, epoch primitives.Epoch, index uint64) blobIdent {
	return blobIdent{root: root, epoch: epoch, index: index}
}

func identForSidecar(sc blocks.VerifiedROBlob) blobIdent {
	return newBlobIdent(sc.BlockRoot(), slots.ToEpoch(sc.Slot()), sc.Index)
}

func (n blobIdent) sszFname() string {
	return fmt.Sprintf("%d.%s", n.index, sszExt)
}

func (n blobIdent) partFname(entropy string) string {
	return fmt.Sprintf("%s-%d.%s", entropy, n.index, partExt)
}

func (n blobIdent) logFields() logrus.Fields {
	return logrus.Fields{
		"root":  fmt.Sprintf("%#x", n.root),
		"epoch": n.epoch,
		"index": n.index,
	}
}

type fsLayout interface {
	dir(n blobIdent) string
	sszPath(n blobIdent) string
	partPath(n blobIdent, entropy string) string
	iterateIdents(before primitives.Epoch) (*identIterator, error)
	ident(root [32]byte, idx uint64) (blobIdent, error)
	dirIdent(root [32]byte) (blobIdent, error)
	summary(root [32]byte) BlobStorageSummary
	notify(ident blobIdent) error
	pruneBefore(before primitives.Epoch) (*pruneSummary, error)
	remove(ident blobIdent) (int, error)
}

func newLayout(name string, fs afero.Fs, cache *blobStorageCache, pruner *blobPruner) (fsLayout, error) {
	switch name {
	case LayoutNameFlat:
		return newFlatLayout(fs, cache, pruner), nil
	case LayoutNameByEpoch:
		return newPeriodicEpochLayout(fs, cache, pruner), nil
	default:
		return nil, errors.Wrapf(errInvalidLayoutName, "name=%s", name)
	}
}

func warmCache(l fsLayout, cache *blobStorageCache) error {
	iter, err := l.iterateIdents(0)
	if err != nil {
		return errors.Wrap(errCacheWarmFailed, err.Error())
	}
	for ident, err := iter.next(); err != io.EOF; ident, err = iter.next() {
		if errors.Is(err, errIdentFailure) {
			idf := &identificationError{}
			if errors.As(err, &idf) {
				log.WithFields(idf.LogFields()).WithError(err).Error("Failed to cache blob data for path")
			}
			continue
		}
		if err != nil {
			return errors.Wrapf(errCacheWarmFailed, "failed to populate blob data cache err=%s", err.Error())
		}
		if err := cache.ensure(ident); err != nil {
			return errors.Wrapf(errCacheWarmFailed, "failed to write cache entry for %s, err=%s", l.sszPath(ident), err.Error())
		}
	}
	return nil
}

func migrateLayout(fs afero.Fs, from, to fsLayout, cache *blobStorageCache) error {
	start := time.Now()
	iter, err := from.iterateIdents(0)
	if err != nil {
		return errors.Wrapf(errMigrationFailure, "failed to iterate legacy structure while migrating blobs, err=%s", err.Error())
	}
	lastMoved := ""
	parentDirs := make(map[string]bool) // this map should have < 65k keys by design
	moved := 0
	for ident, err := iter.next(); err != io.EOF; ident, err = iter.next() {
		if err != nil {
			if errors.Is(err, errIdentFailure) {
				idf := &identificationError{}
				if errors.As(err, &idf) {
					log.WithFields(idf.LogFields()).WithError(err).Error("Failed to migrate blob path")
				}
				continue
			}
			return errors.Wrapf(errMigrationFailure, "failed to iterate previous layout structure while migrating blobs, err=%s", err.Error())
		}
		src := from.dir(ident)
		target := to.dir(ident)
		if src != lastMoved {
			targetParent := filepath.Dir(target)
			if targetParent != "" && targetParent != "." && !parentDirs[targetParent] {
				if err := fs.MkdirAll(targetParent, directoryPermissions()); err != nil {
					return errors.Wrapf(errMigrationFailure, "failed to make enclosing path before moving %s to %s", src, target)
				}
				parentDirs[targetParent] = true
			}
			if err := fs.Rename(src, target); err != nil {
				return errors.Wrapf(errMigrationFailure, "could not rename %s to %s", src, target)
			}
			moved += 1
			lastMoved = src
		}
		if err := cache.ensure(ident); err != nil {
			return errors.Wrapf(errMigrationFailure, "could not cache path %s, err=%s", to.sszPath(ident), err.Error())
		}
	}
	if moved > 0 {
		log.WithField("dirsMoved", moved).WithField("elapsed", time.Since(start)).
			Info("Blob filesystem migration complete.")
	}
	return nil
}

type pruneSummary struct {
	blobsPruned    int
	failedRemovals []string
}

func (s pruneSummary) LogFields() logrus.Fields {
	return logrus.Fields{
		"blobsPruned":    s.blobsPruned,
		"failedRemovals": len(s.failedRemovals),
	}
}

func pruneBefore(before primitives.Epoch, l fsLayout) (map[primitives.Epoch]*pruneSummary, error) {
	sums := make(map[primitives.Epoch]*pruneSummary)
	iter, err := l.iterateIdents(before)
	if err != nil {
		return nil, errors.Wrap(err, "failed to iterate blob paths for pruning")
	}

	// We will get an ident for each index, but want to prune all indexes for the given root together.
	var lastIdent blobIdent
	for ident, err := iter.next(); err != io.EOF; ident, err = iter.next() {
		if err != nil {
			if errors.Is(err, errIdentFailure) {
				idf := &identificationError{}
				if errors.As(err, &idf) {
					log.WithFields(idf.LogFields()).WithError(err).Error("Failed to prune blob path due to identification errors")
				}
				continue
			}
			log.WithError(err).Error("encountered unhandled error during pruning")
			return nil, errors.Wrap(errPruneFailed, err.Error())
		}
		if ident.epoch >= before {
			continue
		}
		if lastIdent.root != ident.root {
			pruneOne(lastIdent, l, sums)
			lastIdent = ident
		}
	}
	// handle the final ident
	pruneOne(lastIdent, l, sums)

	return sums, nil
}

func pruneOne(ident blobIdent, l fsLayout, sums map[primitives.Epoch]*pruneSummary) {
	// Skip pruning the n-1 ident if we're on the first real ident (lastIdent will be zero value).
	if ident.root == params.BeaconConfig().ZeroHash {
		return
	}
	_, ok := sums[ident.epoch]
	if !ok {
		sums[ident.epoch] = &pruneSummary{}
	}
	s := sums[ident.epoch]
	removed, err := l.remove(ident)
	if err != nil {
		s.failedRemovals = append(s.failedRemovals, l.dir(ident))
		log.WithField("root", fmt.Sprintf("%#x", ident.root)).Error("Failed to delete blob directory for root")
	}
	s.blobsPruned += removed
}
