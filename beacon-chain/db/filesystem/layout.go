package filesystem

import (
	"context"
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
	errInvalidRootString      = errors.New("Could not parse hex string as a [32]byte")
	errInvalidDirectoryLayout = errors.New("Could not parse blob directory path")
)

type migratableLayout interface {
	dir(n blobNamer) string
	sszPath(n blobNamer) string
	partPath(n blobNamer, entropy string) string
	iterateNamers() (*namerIterator, error)
}

type fsLayout interface {
	migratableLayout
	dirNamer(root [32]byte) (blobNamer, error)
	namer(root [32]byte, idx uint64) (blobNamer, error)
	summary(root [32]byte) BlobStorageSummary
	initialize() error
	notify(sidecar blocks.VerifiedROBlob) error
	pruneBefore(before primitives.Epoch) (pruneSummary, error)
	waitForSummarizer(ctx context.Context) (BlobStorageSummarizer, error)
}

type blobNamer struct {
	version string
	root    [32]byte
	epoch   primitives.Epoch
	index   uint64
	err     error
}

func newBlobNamer(root [32]byte, epoch primitives.Epoch, index uint64) blobNamer {
	return blobNamer{root: root, epoch: epoch, index: index}
}

func namerForSidecar(sc blocks.VerifiedROBlob) blobNamer {
	return newBlobNamer(sc.BlockRoot(), slots.ToEpoch(sc.Slot()), sc.Index)
}

func (n blobNamer) sszFname() string {
	return fmt.Sprintf("%d.%s", n.index, sszExt)
}

func (n blobNamer) partFname(entropy string) string {
	return fmt.Sprintf("%s-%d.%s", entropy, n.index, partExt)
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
var _ fsLayout = &periodicEpochLayout{}

type flatRootLayout struct {
	fs afero.Fs
}

func (l *flatRootLayout) iterateNamers() (*namerIterator, error) {
	entries, err := listDir(l.fs, ".")
	if err != nil {
		return nil, errors.Wrapf(err, "could not list root directory")
	}
	slotAndIndex := &readSlotOncePerRoot{fs: l.fs}
	return &namerIterator{
		fs: l.fs,
		levels: []layoutLevel{
			{namerUpdater: rootNamerUpdater, filter: isRootDir},
			{namerUpdater: slotAndIndex.namerUpdater, filter: isSszFile}},
		entries: entries,
	}, nil
}

func (l *flatRootLayout) dir(n blobNamer) string {
	return rootToString(n.root)
}

func (l *flatRootLayout) sszPath(n blobNamer) string {
	return path.Join(l.dir(n), n.sszFname())
}

func (l *flatRootLayout) partPath(n blobNamer, entropy string) string {
	return path.Join(l.dir(n), n.partFname(entropy))
}

type periodicEpochLayout struct {
	fs     afero.Fs
	cache  *blobStorageCache
	pruner *blobPruner
}

func (l *periodicEpochLayout) waitForSummarizer(ctx context.Context) (BlobStorageSummarizer, error) {
	if err := l.cache.waitForReady(ctx); err != nil {
		return nil, err
	}
	return l.cache, nil
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

func (l *periodicEpochLayout) iterateNamers() (*namerIterator, error) {
	// iterate root, which should have directories named by "period"
	entries, err := listDir(l.fs, periodicEpochBaseDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list %s", periodicEpochBaseDir)
	}

	return &namerIterator{
		fs:   l.fs,
		path: periodicEpochBaseDir,
		levels: []layoutLevel{
			{namerUpdater: noopNamerUpdater, filter: filterNoop},  // no info to extract from "period" level
			{namerUpdater: epochNamerUpdater, filter: filterNoop}, // extract epoch from path, todo: numeric check filter?
			{namerUpdater: rootNamerUpdater, filter: isRootDir},   // extract root from path
			{namerUpdater: indexNamerUpdater, filter: isSszFile},  // extract index from filename
		},
		entries: entries,
	}, nil
}

func (l *periodicEpochLayout) namer(root [32]byte, idx uint64) (blobNamer, error) {
	return l.cache.namerForIdx(root, idx)
}

func (l *periodicEpochLayout) dirNamer(root [32]byte) (blobNamer, error) {
	return l.cache.namerForRoot(root)
}

func (l *periodicEpochLayout) summary(root [32]byte) BlobStorageSummary {
	return l.cache.Summary(root)
}

func (l *periodicEpochLayout) dir(n blobNamer) string {
	return filepath.Join(periodicEpochBaseDir, l.period(n), fmt.Sprintf("%d", n.epoch), rootToString(n.root))
}

func (l *periodicEpochLayout) period(n blobNamer) string {
	period := n.epoch / params.BeaconConfig().MinEpochsForBlobsSidecarsRequest
	return fmt.Sprintf("%d", period)
}

func (l *periodicEpochLayout) sszPath(n blobNamer) string {
	return filepath.Join(l.dir(n), n.sszFname())
}

func (l *periodicEpochLayout) partPath(n blobNamer, entropy string) string {
	return path.Join(l.dir(n), n.partFname(entropy))
}

func (l *periodicEpochLayout) pruneBefore(before primitives.Epoch) (pruneSummary, error) {
	return pruneSummary{}, nil
}

func warmCache(l fsLayout, cache *blobStorageCache) error {
	iter, err := l.iterateNamers()
	if err != nil {
		return errors.Wrap(errCacheWarmFailed, err.Error())
	}
	for namer, err := iter.next(); err != io.EOF; namer, err = iter.next() {
		if err != nil {
			return errors.Wrapf(errMigrationFailure, "failed to iterate legacy structure while migrating blobs, err=%s", err.Error())
		}
		if err := cache.ensure(namer.root, namer.epoch, namer.index); err != nil {
			return errors.Wrapf(errCacheWarmFailed, "failed to write cache entry for %s, err=%s", l.sszPath(namer), err.Error())
		}
	}
	return nil
}

func migrateLayout(fs afero.Fs, from, to migratableLayout, cache *blobStorageCache) error {
	start := time.Now()
	iter, err := from.iterateNamers()
	if err != nil {
		return errors.Wrapf(errMigrationFailure, "failed to iterate legacy structure while migrating blobs, err=%s", err.Error())
	}
	lastMoved := ""
	parentDirs := make(map[string]bool) // this map should have < 65k keys by design
	moved := 0
	for namer, err := iter.next(); err != io.EOF; namer, err = iter.next() {
		if err != nil {
			return errors.Wrapf(errMigrationFailure, "failed to iterate legacy structure while migrating blobs, err=%s", err.Error())
		}
		src := from.dir(namer)
		target := to.dir(namer)
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
		if err := cache.ensure(namer.root, namer.epoch, namer.index); err != nil {
			return errors.Wrapf(errMigrationFailure, "could not cache path %s, err=%s", to.sszPath(namer), err.Error())
		}
	}
	if moved > 0 {
		log.WithField("dirsMoved", moved).WithField("elapsed", time.Since(start)).
			Info("Blob filesystem migration complete.")
	}
	return nil
}

/*
type hexPrefixLayout struct{}

func (l *hexPrefixLayout) initialize(fs afero.Fs) error {
	return fs.MkdirAll(hexPrefixBaseDir, directoryPermissions)
}

func (l *hexPrefixLayout) iterateNamers(ctx context.Context) (chan blobNamer, error) {
	return make(chan blobNamer), nil
}

func (l *hexPrefixLayout) dir(n blobNamer) string {
	rs := rootToString(n.root)
	parentDir := oneBytePrefix(rs)
	return filepath.Join(parentDir, rs)
}

func (l *hexPrefixLayout) Path(n blobNamer) string {
	return path.Join(l.dir(n), fmt.Sprintf("%d.%s", n.index, sszExt))
}

func (l *hexPrefixLayout) partPath(n blobNamer, entropy string) string {
	return path.Join(l.dir(n), fmt.Sprintf("%s-%d.%s", entropy, n.index, partExt))
}

var _ migratableLayout = &hexPrefixLayout{}


func iteratePeriodicLayout(ctx context.Context, periods []string, fs afero.Fs, c chan blobNamer) {
	defer close(c)
	// example layout: blobs/by-epoch/66/273848/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz
	// The caller lists blobs/by-epoch and passes in the result
	// following the above example, one of the values in 'periods' would be '66'
	for _, period := range periods {
		periodDir := filepath.Join(periodicEpochBaseDir, period)
		epochs, err := listDir(fs, periodDir)
		if err != nil {
			c <- blobNamer{err: errors.Wrapf(err, "failed to list %s", periodDir)}
			return
		}
		for _, epochStr := range epochs {
			// following the above example, one of the values in 'epochs' would be '273848'
			epochDir := filepath.Join(periodDir, epochStr)
			epoch, err := epochFromPath(epochDir)
			if err != nil {
				c <- blobNamer{err: errors.Wrapf(err, "failed to decode epoch from dir %s", epochDir)}
				return
			}
			roots, err := listDir(fs, epochDir)
			if err != nil {
				c <- blobNamer{err: errors.Wrapf(err, "failed to list %s", epochDir)}
				return
			}
			for _, rootStr := range roots {
				// following the above example, one of the values in 'roots' would be '0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb'
				rootDir := filepath.Join(epochDir, rootStr)
				root, err := rootFromPath(rootDir)
				if err != nil {
					c <- blobNamer{err: errors.Wrapf(err, "failed to decode root bytes from dir %s", rootDir)}
					return
				}
				fnames, err := listDir(fs, rootDir)
				if err != nil {
					c <- blobNamer{err: errors.Wrapf(err, "failed to list %s", rootDir)}
					return
				}
				for _, fname := range fnames {
					// following the above example, one of the values in 'fnames' would be '0.ssz'
					if !isSszFile(fname) {
						continue
					}
					if ctx.Err() != nil {
						return
					}
					idx, err := idxFromPath(fname)
					if err != nil {
						c <- blobNamer{err: errors.Wrapf(err, "failed to decode blob index from dir %s", filepath.Join(rootDir, fname))}
						return
					}
					c <- newBlobNamer(root, epoch, idx)
				}
			}
		}
	}
}

func iterateLegacy(ctx context.Context, entries []string, fs afero.Fs, c chan blobNamer) {
	defer close(c)
	for _, dir := range entries {
		if !isRootDir(dir) {
			continue
		}
		root, err := rootFromPath(dir)
		if err != nil {
			log.WithField("directory", dir).Error("Skipping directory with unparsable root in path.")
			continue
		}
		files, err := listDir(fs, dir)
		if err != nil {
			log.WithField("directory", dir).Error("Error listing blob storage directory.")
			continue
		}
		var epoch *primitives.Epoch
		for _, fname := range files {
			if ctx.Err() != nil {
				return
			}
			if !isSszFile(fname) {
				continue
			}
			fullPath := filepath.Join(dir, fname)
			idx, err := idxFromPath(fullPath)
			if err != nil {
				log.WithField("path", fullPath).Error("Could not determine index for file")
			}
			namer := newBlobNamer(root, 0, idx)
			if epoch == nil {
				slot, err := slotFromFile(fullPath, fs)
				if err != nil {
					namer.err = errors.Wrapf(err, "error reading slot from file %s", filepath.Join(dir, fname))
					c <- namer
					continue
				}
				e := slots.ToEpoch(slot)
				epoch = &e
			}
			namer.epoch = *epoch
			c <- namer
		}
	}
}


*/
