package filesystem

import (
	"context"
	"encoding/binary"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const retentionBuffer primitives.Epoch = 2
const bytesPerSidecar = 131928

var (
	errPruningFailures = errors.New("blobs could not be pruned for some roots")
	errNotBlobSSZ      = errors.New("not a blob ssz file")
)

type blobPruner struct {
	sync.Mutex
	prunedBefore atomic.Uint64
	windowSize   primitives.Slot
	cache        *blobStorageCache
	cacheReady   chan struct{}
	warmed       bool
	fs           afero.Fs
}

type prunerOpt func(*blobPruner) error

func withWarmedCache() prunerOpt {
	return func(p *blobPruner) error {
		return p.warmCache()
	}
}

func newBlobPruner(fs afero.Fs, retain primitives.Epoch, opts ...prunerOpt) (*blobPruner, error) {
	r, err := slots.EpochStart(retain + retentionBuffer)
	if err != nil {
		return nil, errors.Wrap(err, "could not set retentionSlots")
	}
	cw := make(chan struct{})
	p := &blobPruner{fs: fs, windowSize: r, cache: newBlobStorageCache(), cacheReady: cw}
	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// notify updates the pruner's view of root->blob mappings. This allows the pruner to build a cache
// of root->slot mappings and decide when to evict old blobs based on the age of present blobs.
func (p *blobPruner) notify(root [32]byte, latest primitives.Slot, idx uint64) error {
	if err := p.cache.ensure(rootString(root), latest, idx); err != nil {
		return err
	}
	pruned := uint64(windowMin(latest, p.windowSize))
	if p.prunedBefore.Swap(pruned) == pruned {
		return nil
	}
	go func() {
		p.Lock()
		defer p.Unlock()
		if err := p.prune(primitives.Slot(pruned)); err != nil {
			log.WithError(err).Errorf("Failed to prune blobs from slot %d", latest)
		}
	}()
	return nil
}

func windowMin(latest, offset primitives.Slot) primitives.Slot {
	// Safely compute the first slot in the epoch for the latest slot
	latest = latest - latest%params.BeaconConfig().SlotsPerEpoch
	if latest < offset {
		return 0
	}
	return latest - offset
}

func (p *blobPruner) warmCache() error {
	p.Lock()
	defer p.Unlock()
	if err := p.prune(0); err != nil {
		return err
	}
	if !p.warmed {
		p.warmed = true
		close(p.cacheReady)
	}
	return nil
}

func (p *blobPruner) waitForCache(ctx context.Context) (*blobStorageCache, error) {
	select {
	case <-p.cacheReady:
		return p.cache, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Prune prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpochs+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (p *blobPruner) prune(pruneBefore primitives.Slot) error {
	start := time.Now()
	totalPruned, totalErr := 0, 0
	// Customize logging/metrics behavior for the initial cache warmup when slot=0.
	// We'll never see a prune request for slot 0, unless this is the initial call to warm up the cache.
	if pruneBefore == 0 {
		defer func() {
			log.WithField("duration", time.Since(start).String()).Debug("Warmed up pruner cache")
		}()
	} else {
		defer func() {
			log.WithFields(logrus.Fields{
				"upToEpoch":    slots.ToEpoch(pruneBefore),
				"duration":     time.Since(start).String(),
				"filesRemoved": totalPruned,
			}).Debug("Pruned old blobs")
			blobsPrunedCounter.Add(float64(totalPruned))
		}()
	}

	entries, err := listDir(p.fs, ".")
	if err != nil {
		return errors.Wrap(err, "unable to list root blobs directory")
	}
	dirs := filter(entries, filterRoot)
	for _, dir := range dirs {
		pruned, err := p.tryPruneDir(dir, pruneBefore)
		if err != nil {
			totalErr += 1
			log.WithError(err).WithField("directory", dir).Error("Unable to prune directory")
		}
		totalPruned += pruned
	}

	if totalErr > 0 {
		return errors.Wrapf(errPruningFailures, "pruning failed for %d root directories", totalErr)
	}
	return nil
}

func shouldRetain(slot, pruneBefore primitives.Slot) bool {
	return slot >= pruneBefore
}

func (p *blobPruner) tryPruneDir(dir string, pruneBefore primitives.Slot) (int, error) {
	root := rootFromDir(dir)
	slot, slotCached := p.cache.slot(root)
	// Return early if the slot is cached and doesn't need pruning.
	if slotCached && shouldRetain(slot, pruneBefore) {
		return 0, nil
	}

	// entries will include things that aren't ssz files, like dangling .part files. We need these to
	// completely clean up the directory.
	entries, err := listDir(p.fs, dir)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to list blobs in directory %s", dir)
	}
	// scFiles filters the dir listing down to the ssz encoded BlobSidecar files. This allows us to peek
	// at the first one in the list to figure out the slot.
	scFiles := filter(entries, filterSsz)
	if len(scFiles) == 0 {
		log.WithField("dir", dir).Warn("Pruner ignoring directory with no blob files")
		return 0, nil
	}
	if !slotCached {
		slot, err = slotFromFile(path.Join(dir, scFiles[0]), p.fs)
		if err != nil {
			return 0, errors.Wrapf(err, "slot could not be read from blob file %s", scFiles[0])
		}
		for i := range scFiles {
			idx, err := idxFromPath(scFiles[i])
			if err != nil {
				return 0, errors.Wrapf(err, "index could not be determined for blob file %s", scFiles[i])
			}
			if err := p.cache.ensure(root, slot, idx); err != nil {
				return 0, errors.Wrapf(err, "could not update prune cache for blob file %s", scFiles[i])
			}
		}
		if shouldRetain(slot, pruneBefore) {
			return 0, nil
		}
	}

	removed := 0
	for _, fname := range entries {
		fullName := path.Join(dir, fname)
		if err := p.fs.Remove(fullName); err != nil {
			return removed, errors.Wrapf(err, "unable to remove %s", fullName)
		}
		// Don't count other files that happen to be in the dir, like dangling .part files.
		if filterSsz(fname) {
			removed += 1
		}
		// Log a warning whenever we clean up a .part file
		if filterPart(fullName) {
			log.WithField("file", fullName).Warn("Deleting abandoned blob .part file")
		}
	}
	if err := p.fs.Remove(dir); err != nil {
		return removed, errors.Wrapf(err, "unable to remove blob directory %s", dir)
	}

	p.cache.evict(rootFromDir(dir))
	return len(scFiles), nil
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

func rootFromDir(dir string) string {
	return filepath.Base(dir) // end of the path should be the blob directory, named by hex encoding of root
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

var dotSszExt = "." + sszExt
var dotPartExt = "." + partExt

func filterSsz(s string) bool {
	return filepath.Ext(s) == dotSszExt
}

func filterPart(s string) bool {
	return filepath.Ext(s) == dotPartExt
}
