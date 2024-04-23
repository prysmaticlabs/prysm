package filesystem

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/sirupsen/logrus"
)

const retentionBuffer primitives.Epoch = 2

var (
	errPruningFailures = errors.New("blobs could not be pruned for some roots")
	errNotBlobSSZ      = errors.New("not a blob ssz file")
)

type blobPruner struct {
	mu              sync.Mutex
	prunedBefore    atomic.Uint64
	retentionPeriod primitives.Epoch
}

func newBlobPruner(retain primitives.Epoch) *blobPruner {
	p := &blobPruner{retentionPeriod: retain + retentionBuffer}
	return p
}

func (p *blobPruner) notify(latest primitives.Epoch, layout fsLayout) {
	floor := periodFloor(latest, p.retentionPeriod)
	if primitives.Epoch(p.prunedBefore.Swap(uint64(floor))) == floor {
		// Only trigger pruning if the atomic swap changed the previous value of prunedBefore.
		return
	}
	go func() {
		p.mu.Lock()
		start := time.Now()
		defer p.mu.Unlock()
		sum, err := layout.PruneBefore(floor)
		if err != nil {
			log.WithError(err).WithFields(sum.LogFields()).Warn("Encountered errors during blob pruning.")
		}
		log.WithFields(logrus.Fields{
			"upToEpoch":    floor,
			"duration":     time.Since(start).String(),
			"filesRemoved": sum.blobsPruned,
		}).Debug("Pruned old blobs")
		blobsPrunedCounter.Add(float64(sum.blobsPruned))
	}()
}

func periodFloor(latest, period primitives.Epoch) primitives.Epoch {
	if latest < period {
		return 0
	}
	return latest - period
}

/*
func (p *blobPruner) tryPruneDir(dir string, pruneBefore primitives.Slot) (int, error) {
	root, err := rootFromDir(dir)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid directory, could not parse subdir as root %s", dir)
	}
	epoch, slotCached := p.cache.epoch(root)
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

	p.cache.evict(root)
	return len(scFiles), nil
}

*/
