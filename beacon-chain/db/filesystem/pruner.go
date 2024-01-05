package filesystem

import (
	"encoding/binary"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const retentionBuffer primitives.Epoch = 2

var (
	errBlobSlotUnknown = errors.New("could not determine blob slot from files in storage")
	errPruningFailures = errors.New("blobs could not be pruned for some roots")
)

type blobPruner struct {
	sync.Mutex
	slotMap      *slotForRoot
	retain       primitives.Slot
	fs           afero.Fs
	prunedBefore atomic.Uint64
}

func newblobPruner(fs afero.Fs, retain primitives.Epoch) (*blobPruner, error) {
	r, err := slots.EpochStart(retain + retentionBuffer)
	if err != nil {
		return nil, errors.Wrap(err, "could not set retentionSlots")
	}
	return &blobPruner{fs: fs, retain: r, slotMap: newSlotForRoot()}, nil
}

// tryPrune checks whether we should prune and then calls prune in a goroutine.
func (p *blobPruner) try(root [32]byte, latest primitives.Slot) {
	p.slotMap.ensure(rootString(root), latest)
	pruned := uint64(pruneBefore(latest, p.retain))
	if p.prunedBefore.Swap(pruned) == pruned {
		return
	}
	go func() {
		if err := p.prune(primitives.Slot(pruned)); err != nil {
			log.WithError(err).Errorf("failed to prune blobs from slot %d", latest)
		}
	}()
}

func pruneBefore(latest primitives.Slot, offset primitives.Slot) primitives.Slot {
	// Safely compute the first slot in the epoch for the latest slot
	latest = latest - latest%params.BeaconConfig().SlotsPerEpoch
	if latest < offset {
		return 0
	}
	return latest - offset
}

// Prune prunes blobs in the base directory based on the retention epoch.
// It deletes blobs older than currentEpoch - (retentionEpochs+bufferEpochs).
// This is so that we keep a slight buffer and blobs are deleted after n+2 epochs.
func (p *blobPruner) prune(pruneBefore primitives.Slot) error {
	log.Debug("Pruning old blobs")
	start := time.Now()
	totalPruned, totalErr := 0, 0
	defer func() {
		log.WithFields(log.Fields{
			"lastPrunedEpoch":   slots.ToEpoch(pruneBefore),
			"pruneTime":         time.Since(start).String(),
			"numberBlobsPruned": totalPruned,
		}).Debug("Pruned old blobs")
		blobsPrunedCounter.Add(float64(totalPruned))
	}()

	dirs, err := p.listDir(".", filterRoot)
	if err != nil {
		return errors.Wrap(err, "unable to list root blobs directory")
	}
	for _, dir := range dirs {
		pruned, err := p.tryPruneDir(dir, pruneBefore)
		if err != nil {
			totalErr += 1
			log.WithError(err).WithField("directory", dir).Error("unable to prune directory")
		}
		totalPruned += pruned
	}

	if totalErr > 0 {
		return errors.Wrapf(errPruningFailures, "pruning failed for %d root directories", totalErr)
	}
	return nil
}

// directoryMeta tries a few different ways to determine the slot for the given directory.
// The seconds argument will be nil if the function did not need to list the directory, or
// non-nil with a list of files if it did.
func (p *blobPruner) directoryMeta(dir string) (primitives.Slot, []string, error) {
	root := filepath.Base(dir) // end of the path should be the blob directory, named by hex encoding of root
	// First try the cheap map lookup.
	slot, ok := p.slotMap.slot(root)
	if ok {
		return slot, nil, nil
	}

	// Next try constructing the path to the zero index blob, which will always be present unless
	// the blob directory has been damaged by something like a restart during RemoveAll.
	slot, err := slotFromFile(filepath.Join(dir, "0."+sszExt), p.fs)
	if err == nil {
		p.slotMap.ensure(root, slot)
		return slot, nil, nil
	}

	// Fall back if getting the slot from index zero failed -- look for any ssz file.
	files, err := p.listDir(dir, filterSsz)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "failed to list blobs in directory %s", dir)
	}
	if len(files) == 0 {
		return 0, files, errors.Wrapf(errBlobSlotUnknown, "contained no blob files")
	}
	slot, err = slotFromFile(files[0], p.fs)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "slot could not be read from blob file %s", files[0])
	}
	p.slotMap.ensure(root, slot)
	return slot, files, nil
}

// tryPruneDir will delete the directory of blobs if the blob slot is outside the
// retention period. We determine the slot by looking at the first blob in the directory.
func (p *blobPruner) tryPruneDir(dir string, pruneBefore primitives.Slot) (int, error) {
	slot, files, err := p.directoryMeta(dir)
	if err != nil {
		return 0, errors.Wrapf(err, "could not determine slot for directory %s", dir)
	}
	if slot >= pruneBefore {
		return 0, nil
	}

	if len(files) == 0 {
		files, err = p.listDir(dir, filterSsz)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to list blobs in directory %s", dir)
		}
	}
	if err = p.fs.RemoveAll(dir); err != nil {
		return 0, errors.Wrapf(err, "failed to delete blobs in %s", dir)
	}
	return len(files), nil
}

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

func (p *blobPruner) listDir(dir string, filter func(string) bool) ([]string, error) {
	top, err := p.fs.Open(dir)
	defer func() {
		if err := top.Close(); err != nil {
			log.WithError(err).Errorf("Could not close file %s", dir)
		}
	}()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open directory descriptor")
	}
	dirs, err := top.Readdirnames(-1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read directory listing")
	}
	if filter != nil {
		filtered := make([]string, 0, len(dirs))
		for i := range dirs {
			if filter(dirs[i]) {
				filtered = append(filtered, dirs[i])
			}
		}
		return filtered, nil
	}
	return dirs, nil
}

func filterRoot(s string) bool {
	return strings.HasPrefix(s, "0x")
}

func filterSsz(s string) bool {
	return filepath.Ext(s) == sszExt
}

func newSlotForRoot() *slotForRoot {
	return &slotForRoot{
		cache: make(map[string]primitives.Slot, params.BeaconConfig().MinEpochsForBlobsSidecarsRequest*fieldparams.SlotsPerEpoch),
	}
}

type slotForRoot struct {
	sync.RWMutex
	cache map[string]primitives.Slot
}

func (s *slotForRoot) ensure(key string, slot primitives.Slot) {
	s.Lock()
	defer s.Unlock()
	s.cache[key] = slot
}

func (s *slotForRoot) slot(key string) (primitives.Slot, bool) {
	s.RLock()
	defer s.RUnlock()
	slot, ok := s.cache[key]
	return slot, ok
}

func (s *slotForRoot) evict(key string) {
	s.Lock()
	defer s.Unlock()
	delete(s.cache, key)
}
