package das

import (
	"context"
	"sync"

	errors "github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// BlockingStore wraps NewCachingDBVerifiedStore and blocks until IsDataAvailable is ready.
// If the context given to IsDataAvailable is cancelled, the result of IsDataAvailable will be ctx.Error().
type BlockingStore struct {
	notif *idxNotifiers
	s     AvailabilityStore
}

func (bs *BlockingStore) PersistOnceCommitted(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) []*ethpb.BlobSidecar {
	if len(sc) < 1 {
		return nil
	}
	seen := bs.notif.ensure(keyFromSidecar(sc[0]))
	persisted := bs.s.PersistOnceCommitted(ctx, current, sc...)
	for i := range persisted {
		seen <- persisted[i].Index
	}
	return persisted
}

func (bs *BlockingStore) IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	key := keyFromBlock(b)
	for {
		err := bs.s.IsDataAvailable(ctx, current, b)
		if err == nil {
			return nil
		}
		mie := &MissingIndicesError{}
		if !errors.As(err, mie) {
			return err
		}
		waitFor := make(map[uint64]struct{})
		for _, m := range mie.Missing() {
			waitFor[m] = struct{}{}
		}
		if err := waitForIndices(ctx, bs.notif.ensure(key), waitFor); err != nil {
			return err
		}
		bs.notif.reset(key)
	}
}

func waitForIndices(ctx context.Context, idxSeen chan uint64, waitFor map[uint64]struct{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case idx := <-idxSeen:
			delete(waitFor, idx)
			if len(waitFor) == 0 {
				return nil
			}
			continue
		}
	}
}

type idxNotifiers struct {
	sync.RWMutex
	entries map[cacheKey]chan uint64
}

func (c *idxNotifiers) ensure(key cacheKey) chan uint64 {
	c.Lock()
	defer c.Unlock()
	e, ok := c.entries[key]
	if !ok {
		e = make(chan uint64, fieldparams.MaxBlobsPerBlock)
		c.entries[key] = e
	}
	return e
}

func (c *idxNotifiers) reset(key cacheKey) {
	c.Lock()
	defer c.Unlock()
	delete(c.entries, key)
}
