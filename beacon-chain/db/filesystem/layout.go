package filesystem

import (
	"fmt"
	"io"
	"path"
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
	rootPrefixLen = 4
	// Full root in directory will be 66 chars, eg:
	// >>> len('0x0002fb4db510b8618b04dc82d023793739c26346a8b02eb73482e24b0fec0555') == 66
	rootStringLen        = 66
	sszExt               = "ssz"
	partExt              = "part"
	periodicEpochBaseDir = "by-epoch"
	hexPrefixBaseDir     = "by-hex-prefix"
)

var (
	errMigrationFailure       = errors.New("unable to migrate blob directory between old and new layout")
	errCacheWarmFailed        = errors.New("failed to warm blob filesystem cache")
	errPruneFailed            = errors.New("failed to prune root")
	errInvalidRootString      = errors.New("Could not parse hex string as a [32]byte")
	errInvalidDirectoryLayout = errors.New("Could not parse blob directory path")
)

type migratableLayout interface {
	dir(n blobIdent) string
	sszPath(n blobIdent) string
	partPath(n blobIdent, entropy string) string
	iterateIdents(before primitives.Epoch) (*identIterator, error)
}

type runtimeLayout interface {
	migratableLayout
	ident(root [32]byte, idx uint64) (blobIdent, error)
	dirIdent(root [32]byte) (blobIdent, error)
	summary(root [32]byte) BlobStorageSummary
	notify(sidecar blocks.VerifiedROBlob) error
	pruneBefore(before primitives.Epoch) (*pruneSummary, error)
	remove(ident blobIdent) (int, error)
}

func warmCache(l runtimeLayout, cache *blobStorageCache) error {
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
		if err := cache.ensure(ident.root, ident.epoch, ident.index); err != nil {
			return errors.Wrapf(errCacheWarmFailed, "failed to write cache entry for %s, err=%s", l.sszPath(ident), err.Error())
		}
	}
	return nil
}

func migrateLayout(fs afero.Fs, from, to migratableLayout, cache *blobStorageCache) error {
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
			return errors.Wrapf(errMigrationFailure, "failed to iterate legacy structure while migrating blobs, err=%s", err.Error())
		}
		src := from.dir(ident)
		target := to.dir(ident)
		if src != lastMoved {
			targetParent := filepath.Dir(target)
			if targetParent != "" && targetParent != "." && !parentDirs[targetParent] {
				if err := fs.MkdirAll(targetParent, directoryPermissions); err != nil {
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
		if err := cache.ensure(ident.root, ident.epoch, ident.index); err != nil {
			return errors.Wrapf(errMigrationFailure, "could not cache path %s, err=%s", to.sszPath(ident), err.Error())
		}
	}
	if moved > 0 {
		log.WithField("dirsMoved", moved).WithField("elapsed", time.Since(start)).
			Info("Blob filesystem migration complete.")
	}
	return nil
}

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

type pruneSummary struct {
	blobsPruned    int
	failedRemovals []string
}

func (s pruneSummary) LogFields() logrus.Fields {
	return logrus.Fields{}
}

func newPeriodicEpochLayout(fs afero.Fs, cache *blobStorageCache, pruner *blobPruner) (*periodicEpochLayout, error) {
	l := &periodicEpochLayout{fs: fs, cache: cache, pruner: pruner}
	if err := l.initialize(); err != nil {
		return nil, err
	}
	return l, nil
}

var _ migratableLayout = &flatRootLayout{}
var _ runtimeLayout = &periodicEpochLayout{}

type periodicEpochLayout struct {
	fs     afero.Fs
	cache  *blobStorageCache
	pruner *blobPruner
}

func (l *periodicEpochLayout) notify(sc blocks.VerifiedROBlob) error {
	epoch := slots.ToEpoch(sc.Slot())
	if err := l.cache.ensure(sc.BlockRoot(), epoch, sc.Index); err != nil {
		return err
	}
	l.pruner.notify(epoch, l)
	return nil
}

func (l *periodicEpochLayout) initialize() error {
	return l.fs.MkdirAll(periodicEpochBaseDir, directoryPermissions)
}

// If before == 0, it won't be used as a filter and all idents will be returned.
func (l *periodicEpochLayout) iterateIdents(before primitives.Epoch) (*identIterator, error) {
	// iterate root, which should have directories named by "period"
	entries, err := listDir(l.fs, periodicEpochBaseDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list %s", periodicEpochBaseDir)
	}

	return &identIterator{
		fs:   l.fs,
		path: periodicEpochBaseDir,
		levels: []layoutLevel{
			{populateIdent: populateNoop, filter: isBeforePeriod(before)},
			{populateIdent: populateEpoch, filter: isBeforeEpoch(before)},
			{populateIdent: populateRoot, filter: isRootDir},  // extract root from path
			{populateIdent: populateIndex, filter: isSszFile}, // extract index from filename
		},
		entries: entries,
	}, nil
}

func (l *periodicEpochLayout) ident(root [32]byte, idx uint64) (blobIdent, error) {
	return l.cache.identForIdx(root, idx)
}

func (l *periodicEpochLayout) dirIdent(root [32]byte) (blobIdent, error) {
	return l.cache.identForRoot(root)
}

func (l *periodicEpochLayout) summary(root [32]byte) BlobStorageSummary {
	return l.cache.Summary(root)
}

func (l *periodicEpochLayout) dir(n blobIdent) string {
	return filepath.Join(l.epochDir(n.epoch), rootToString(n.root))
}

func (l *periodicEpochLayout) epochDir(epoch primitives.Epoch) string {
	return filepath.Join(periodicEpochBaseDir, fmt.Sprintf("%d", periodForEpoch(epoch)), fmt.Sprintf("%d", epoch))
}

func periodForEpoch(epoch primitives.Epoch) primitives.Epoch {
	return epoch / params.BeaconConfig().MinEpochsForBlobsSidecarsRequest
}

func (l *periodicEpochLayout) sszPath(n blobIdent) string {
	return filepath.Join(l.dir(n), n.sszFname())
}

func (l *periodicEpochLayout) partPath(n blobIdent, entropy string) string {
	return path.Join(l.dir(n), n.partFname(entropy))
}

func (l *periodicEpochLayout) pruneBefore(before primitives.Epoch) (*pruneSummary, error) {
	sums := make(map[primitives.Epoch]*pruneSummary)
	iter, err := l.iterateIdents(before)

	rollup := &pruneSummary{}
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

	// Roll up summaries and clean up per-epoch directories.
	for epoch, sum := range sums {
		rollup.blobsPruned += sum.blobsPruned
		rollup.failedRemovals = append(rollup.failedRemovals, sum.failedRemovals...)
		rmdir := l.epochDir(epoch)
		if len(sum.failedRemovals) == 0 {
			if err := l.fs.Remove(rmdir); err != nil {
				log.WithField("dir", rmdir).WithError(err).Error("Failed to remove epoch directory while pruning")
			}
		} else {
			log.WithField("dir", rmdir).WithField("numFailed", len(sum.failedRemovals)).WithError(err).Error("Unable to remove epoch directory due to pruning failures")
		}
	}

	return rollup, nil
}

func (l *periodicEpochLayout) remove(ident blobIdent) (int, error) {
	removed := l.cache.evict(ident.root)
	if err := l.fs.RemoveAll(l.dir(ident)); err != nil {
		return removed, err
	}
	return removed, nil
}

type flatRootLayout struct {
	fs afero.Fs
}

func (l *flatRootLayout) iterateIdents(_ primitives.Epoch) (*identIterator, error) {
	entries, err := listDir(l.fs, ".")
	if err != nil {
		return nil, errors.Wrapf(err, "could not list root directory")
	}
	slotAndIndex := &readSlotOncePerRoot{fs: l.fs}
	return &identIterator{
		fs: l.fs,
		levels: []layoutLevel{
			{populateIdent: populateRoot, filter: isRootDir},
			{populateIdent: slotAndIndex.populateIdent, filter: isSszFile}},
		entries: entries,
	}, nil
}

func (l *flatRootLayout) dir(n blobIdent) string {
	return rootToString(n.root)
}

func (l *flatRootLayout) sszPath(n blobIdent) string {
	return path.Join(l.dir(n), n.sszFname())
}

func (l *flatRootLayout) partPath(n blobIdent, entropy string) string {
	return path.Join(l.dir(n), n.partFname(entropy))
}
