package filesystem

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/spf13/afero"
)

var (
	errMigrationFailure = errors.New("unable to migrate blob directory between old and new layout")
	errCacheWarmFailed  = errors.New("failed to warm blob filesystem cache")
)

type migratableLayout interface {
	Dir(n blobNamer) string
	Path(n blobNamer) string
	PartPath(n blobNamer, entropy string) string
	IterateNamers(ctx context.Context) (chan blobNamer, error)
}

type fsLayout interface {
	migratableLayout
	Initialize() error
	Notify(sidecar blocks.VerifiedROBlob) error
	WarmCache(ctx context.Context) error
}

func newPeriodicEpochLayout() *periodicEpochLayout {
	return &periodicEpochLayout{cache: newBlobStorageCache()}
}

var _ migratableLayout = &flatRootLayout{}
var _ migratableLayout = &periodicEpochLayout{}
var _ fsLayout = &periodicEpochLayout{}

type flatRootLayout struct {
	fs afero.Fs
}

func iterateLegacy(ctx context.Context, entries []string, fs afero.Fs, c chan blobNamer) {
	defer close(c)
	for _, dir := range entries {
		if !filterLegacy(dir) {
			continue
		}
		root, err := rootFromDir(dir)
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
			if !filterSsz(fname) {
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

func (l *flatRootLayout) IterateNamers(ctx context.Context) (chan blobNamer, error) {
	entries, err := listDir(l.fs, ".")
	if err != nil {
		return nil, errors.Wrapf(err, "could not list root directory")
	}
	// Buffer about an epoch's worth of namers to read ahead of other io.
	c := make(chan blobNamer, params.BeaconConfig().SlotsPerEpoch*fieldparams.MaxBlobsPerBlock)
	go iterateLegacy(ctx, entries, l.fs, c)
	return c, nil
}

func (l *flatRootLayout) Initialize() error {
	return nil
}

func (l *flatRootLayout) Dir(n blobNamer) string {
	return rootString(n.root)
}

func (l *flatRootLayout) Path(n blobNamer) string {
	return path.Join(l.Dir(n), fmt.Sprintf("%d.%s", n.index, sszExt))
}

func (l *flatRootLayout) PartPath(n blobNamer, entropy string) string {
	return path.Join(l.Dir(n), fmt.Sprintf("%s-%d.%s", entropy, n.index, partExt))
}

type periodicEpochLayout struct {
	fs     afero.Fs
	cache  *blobStorageCache
	pruner *blobPruner
}

func (l *periodicEpochLayout) Notify(sc blocks.VerifiedROBlob) error {
	if err := l.cache.ensure(sc.BlockRoot(), slots.ToEpoch(sc.Slot()), sc.Index); err != nil {
		return err
	}
	return nil
}

func (l *periodicEpochLayout) Initialize() error {
	return l.fs.MkdirAll(periodicEpochBaseDir, directoryPermissions)
}

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
			epoch, err := epochFromDir(epochDir)
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
				root, err := rootFromDir(rootDir)
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
					if !filterSsz(fname) {
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

func (l *periodicEpochLayout) IterateNamers(ctx context.Context) (chan blobNamer, error) {
	// iterate root, which should have directories named by "period"
	periods, err := listDir(l.fs, periodicEpochBaseDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list %s", periodicEpochBaseDir)
	}
	// Buffer about an epoch's worth of namers to read ahead of other io.
	c := make(chan blobNamer, params.BeaconConfig().SlotsPerEpoch*fieldparams.MaxBlobsPerBlock)
	go iteratePeriodicLayout(ctx, periods, l.fs, c)
	return c, nil
}

func (l *periodicEpochLayout) Dir(n blobNamer) string {
	return filepath.Join(fmt.Sprintf("%s", n.period), fmt.Sprintf("%s", n.epoch), rootString(n.root))
}

func (l *periodicEpochLayout) Path(n blobNamer) string {
	return filepath.Join(l.Dir(n), fmt.Sprintf("%d.%s", n.index, sszExt))
}

func (l *periodicEpochLayout) PartPath(n blobNamer, entropy string) string {
	return path.Join(l.Dir(n), fmt.Sprintf("%s-%d.%s", entropy, n.index, partExt))
}

func (l *periodicEpochLayout) WarmCache(ctx context.Context) error {
	if err := l.Initialize(); err != nil {
		return errors.Wrap(errCacheWarmFailed, err.Error())
	}

	iter, err := l.IterateNamers(ctx)
	if err != nil {
		return errors.Wrap(errMigrationFailure, err.Error())
	}
	for namer := range iter {
		if err := l.cache.ensure(namer.root, namer.epoch, namer.index); err != nil {
			return errors.Wrapf(errMigrationFailure, "failed to write cache entry for %s, err=%s", l.Path(namer), err.Error())
		}
	}

	// run migration after we populate the cache of values that are in the expected layout so we don't have to iterate moved dirs twice.
	return migrateLayout(ctx, l.fs, &flatRootLayout{fs: l.fs}, l, l.cache)
}

func migrateLayout(ctx context.Context, fs afero.Fs, old, new migratableLayout, cache *blobStorageCache) error {
	iter, err := old.IterateNamers(ctx)
	if err != nil {
		return errors.Wrapf(errMigrationFailure, "failed to iterate through legacy directory structure, err=%s", err.Error())
	}
	lastMoved := ""
	parentDirs := make(map[string]bool) // this map should have < 65k keys by design
	for namer := range iter {
		src := old.Dir(namer)
		target := new.Dir(namer)
		if src != lastMoved {
			// TODO: test that filepath.Dir returns the parent when the child is also a dir
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
			lastMoved = src
		}
		if err := cache.ensure(namer.root, namer.epoch, namer.index); err != nil {
			return errors.Wrapf(errMigrationFailure, "could not cache path %s, err=%s", new.Path(namer), err.Error())
		}
	}
	return nil
}

/*
type hexPrefixLayout struct{}

func (l *hexPrefixLayout) Initialize(fs afero.Fs) error {
	return fs.MkdirAll(hexPrefixBaseDir, directoryPermissions)
}

func (l *hexPrefixLayout) IterateNamers(ctx context.Context) (chan blobNamer, error) {
	return make(chan blobNamer), nil
}

func (l *hexPrefixLayout) Dir(n blobNamer) string {
	rs := rootString(n.root)
	parentDir := oneBytePrefix(rs)
	return filepath.Join(parentDir, rs)
}

func (l *hexPrefixLayout) Path(n blobNamer) string {
	return path.Join(l.Dir(n), fmt.Sprintf("%d.%s", n.index, sszExt))
}

func (l *hexPrefixLayout) PartPath(n blobNamer, entropy string) string {
	return path.Join(l.Dir(n), fmt.Sprintf("%s-%d.%s", entropy, n.index, partExt))
}

var _ migratableLayout = &hexPrefixLayout{}
*/
