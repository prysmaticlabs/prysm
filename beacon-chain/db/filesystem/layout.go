package filesystem

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
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
	Dir(n blobNamer) string
	SszPath(n blobNamer) string
	PartPath(n blobNamer, entropy string) string
	IterateNamers(ctx context.Context) (chan blobNamer, error)
}

type fsLayout interface {
	migratableLayout
	DirNamer(root [32]byte) (blobNamer, error)
	Namer(root [32]byte, idx uint64) (blobNamer, error)
	Summary(root [32]byte) BlobStorageSummary
	Initialize() error
	WarmCache(ctx context.Context) error
	Notify(sidecar blocks.VerifiedROBlob) error
	PruneBefore(before primitives.Epoch) (pruneSummary, error)
	WaitForSummarizer(ctx context.Context) (BlobStorageSummarizer, error)
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
	if err := l.Initialize(); err != nil {
		return nil, err
	}
	return l, nil
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

func (l *flatRootLayout) SszPath(n blobNamer) string {
	return path.Join(l.Dir(n), n.sszFname())
}

func (l *flatRootLayout) PartPath(n blobNamer, entropy string) string {
	return path.Join(l.Dir(n), n.partFname(entropy))
}

type periodicEpochLayout struct {
	fs     afero.Fs
	cache  *blobStorageCache
	pruner *blobPruner
}

func (l *periodicEpochLayout) WaitForSummarizer(ctx context.Context) (BlobStorageSummarizer, error) {
	if err := l.cache.waitForReady(ctx); err != nil {
		return nil, err
	}
	return l.cache, nil
}

func (l *periodicEpochLayout) Notify(sc blocks.VerifiedROBlob) error {
	epoch := slots.ToEpoch(sc.Slot())
	if err := l.cache.ensure(sc.BlockRoot(), epoch, sc.Index); err != nil {
		return err
	}
	l.pruner.notify(epoch, l)
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

func (l *periodicEpochLayout) Namer(root [32]byte, idx uint64) (blobNamer, error) {
	return l.cache.namerForIdx(root, idx)
}

func (l *periodicEpochLayout) DirNamer(root [32]byte) (blobNamer, error) {
	return l.cache.namerForRoot(root)
}

func (l *periodicEpochLayout) Summary(root [32]byte) BlobStorageSummary {
	return l.cache.Summary(root)
}

func (l *periodicEpochLayout) Dir(n blobNamer) string {
	return filepath.Join(l.period(n), fmt.Sprintf("%d", n.epoch), rootString(n.root))
}

func (l *periodicEpochLayout) period(n blobNamer) string {
	period := n.epoch / params.BeaconConfig().MinEpochsForBlobsSidecarsRequest
	return fmt.Sprintf("%d", period)
}

func (l *periodicEpochLayout) SszPath(n blobNamer) string {
	return filepath.Join(l.Dir(n), n.sszFname())
}

func (l *periodicEpochLayout) PartPath(n blobNamer, entropy string) string {
	return path.Join(l.Dir(n), n.partFname(entropy))
}

func (l *periodicEpochLayout) WarmCache(ctx context.Context) error {
	defer l.cache.warmComplete()

	iter, err := l.IterateNamers(ctx)
	if err != nil {
		return errors.Wrap(errCacheWarmFailed, err.Error())
	}
	for namer := range iter {
		if err := l.cache.ensure(namer.root, namer.epoch, namer.index); err != nil {
			return errors.Wrapf(errCacheWarmFailed, "failed to write cache entry for %s, err=%s", l.SszPath(namer), err.Error())
		}
	}

	// run migration after we populate the cache of values that are in the expected layout so we don't have to iterate moved dirs twice.
	if err := migrateLayout(ctx, l.fs, &flatRootLayout{fs: l.fs}, l, l.cache); err != nil {
		return err
	}

	return nil
}

func (l *periodicEpochLayout) PruneBefore(before primitives.Epoch) (pruneSummary, error) {
	return pruneSummary{}, nil
}

// TODO: log elapsed time and number of dirs moved (replicate migration.go)
func migrateLayout(ctx context.Context, fs afero.Fs, from, to migratableLayout, cache *blobStorageCache) error {
	iter, err := from.IterateNamers(ctx)
	if err != nil {
		return errors.Wrapf(errMigrationFailure, "failed to iterate through legacy directory structure, err=%s", err.Error())
	}
	lastMoved := ""
	parentDirs := make(map[string]bool) // this map should have < 65k keys by design
	for namer := range iter {
		src := from.Dir(namer)
		target := to.Dir(namer)
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
			return errors.Wrapf(errMigrationFailure, "could not cache path %s, err=%s", to.SszPath(namer), err.Error())
		}
	}
	return nil
}

func idxFromPath(fname string) (uint64, error) {
	fname = path.Base(fname)

	if filepath.Ext(fname) != dotSszExt {
		return 0, errors.Wrap(errNotBlobSSZ, "does not have .ssz extension")
	}
	parts := strings.Split(fname, ".")
	if len(parts) != 2 {
		return 0, errors.Wrap(errNotBlobSSZ, "unexpected filename structure (want <index>.ssz)")
	}
	return strconv.ParseUint(parts[0], 10, 64)
}

func stringToRoot(str string) ([32]byte, error) {
	if len(str) != rootStringLen {
		return [32]byte{}, errors.Wrapf(errInvalidRootString, "incorrect len for input=%s", str)
	}
	slice, err := hexutil.Decode(str)
	if err != nil {
		return [32]byte{}, errors.Wrapf(errInvalidRootString, "input=%s", str)
	}
	return bytesutil.ToBytes32(slice), nil
}

func rootFromDir(dir string) ([32]byte, error) {
	subdir := filepath.Base(dir) // end of the path should be the blob directory, named by hex encoding of root
	root, err := stringToRoot(subdir)
	if err != nil {
		return root, errors.Wrapf(err, "invalid directory, could not parse subdir as root %s", dir)
	}
	return root, nil
}

func epochFromDir(dir string) (primitives.Epoch, error) {
	subdir := filepath.Base(dir)
	epoch, err := strconv.ParseUint(subdir, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(errInvalidDirectoryLayout,
			"failed to decode epoch as uint, err=%s, dir=%s", err.Error(), dir)
	}
	return primitives.Epoch(epoch), nil
}

// Read slot from marshaled BlobSidecar data in the given file. See slotFromBlob for details.
func slotFromFile(file string, fs afero.Fs) (primitives.Slot, error) {
	f, err := fs.Open(file)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.WithError(err).Errorf("Could not close blob file")
		}
	}()
	return slotFromBlob(f)
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

func listDir(fs afero.Fs, dir string) ([]string, error) {
	top, err := fs.Open(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open directory descriptor")
	}
	defer func() {
		if err := top.Close(); err != nil {
			log.WithError(err).Errorf("Could not close file %s", dir)
		}
	}()
	// re the -1 param: "If n <= 0, Readdirnames returns all the names from the directory in a single slice"
	dirs, err := top.Readdirnames(-1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read directory listing")
	}
	return dirs, nil
}

func filter(entries []string, filt func(string) bool) []string {
	filtered := make([]string, 0, len(entries))
	for i := range entries {
		if filt(entries[i]) {
			filtered = append(filtered, entries[i])
		}
	}
	return filtered
}

func filterRoot(s string) bool {
	return strings.HasPrefix(s, "0x")
}

func filterLegacy(s string) bool {
	return filterRoot(s) && len(s) == rootStringLen
}

var dotSszExt = "." + sszExt
var dotPartExt = "." + partExt

func filterSsz(s string) bool {
	return filepath.Ext(s) == dotSszExt
}

func filterPart(s string) bool {
	return filepath.Ext(s) == dotPartExt
}

func rootString(root [32]byte) string {
	return fmt.Sprintf("%#x", root)
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
