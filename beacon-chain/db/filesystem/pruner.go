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

func (p *blobPruner) notify(latest primitives.Epoch, layout runtimeLayout) chan struct{} {
	done := make(chan struct{})
	floor := periodFloor(latest, p.retentionPeriod)
	if primitives.Epoch(p.prunedBefore.Swap(uint64(floor))) >= floor {
		// Only trigger pruning if the atomic swap changed the previous value of prunedBefore.
		close(done)
		return done
	}
	go func() {
		p.mu.Lock()
		start := time.Now()
		defer p.mu.Unlock()
		sum, err := layout.pruneBefore(floor)
		if err != nil {
			log.WithError(err).WithFields(sum.LogFields()).Warn("Encountered errors during blob pruning.")
		}
		log.WithFields(logrus.Fields{
			"upToEpoch":    floor,
			"duration":     time.Since(start).String(),
			"filesRemoved": sum.blobsPruned,
		}).Debug("Pruned old blobs")
		blobsPrunedCounter.Add(float64(sum.blobsPruned))
		close(done)
	}()
	return done
}

func periodFloor(latest, period primitives.Epoch) primitives.Epoch {
	if latest < period {
		return 0
	}
	return latest - period
}
