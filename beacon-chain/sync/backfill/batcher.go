package backfill

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

var ErrRetryLimitExceeded = errors.New("Unable to retrieve backfill batch")

type batcher struct {
	nWorkers int
	size     primitives.Slot
	su       *StatusUpdater
	todo     chan batch
	done     chan batch
	errc     chan error
	// outstanding is keyed by the id of the batch that is relied on
	// ie if batch id 2 relies on batch id 1, and 1 is head
	outstanding map[batchId]*batch
	nextId      batchId
	lastId      batchId
}

func (br *batcher) run(ctx context.Context) {
	status := br.su.Status()
	// Set min at bottom of backfill range. Add 1 because range is inclusive.
	min := primitives.Slot(status.LowSlot) + 1
	initial := br.next(min, primitives.Slot(status.HighSlot))
	br.nextId, br.lastId = initial.id(), initial.id()
	br.outstanding[initial.id()] = &initial
	br.todo <- initial
	for {
		for i := 0; i < br.nWorkers-len(br.outstanding); i++ {
			last := br.outstanding[br.lastId]
			newLast := br.next(min, last.begin)
			br.outstanding[newLast.id()] = &newLast
			br.lastId = newLast.id()
			br.todo <- newLast
		}
		select {
		case b := <-br.done:
			if err := br.completeBatch(b); err != nil {
				br.errc <- err
			}
		case <-ctx.Done():
			return
		}
	}
}

func (br *batcher) completeBatch(b batch) error {
	// if the batch failed, send it back to the work queue.
	// we have no limit on the number of retries, because all batches are necessary.
	if b.err != nil {
		b.err = nil
		br.outstanding[b.id()] = &b
		br.todo <- b
		return nil
	}

	br.outstanding[b.id()] = &b
	if err := br.includeCompleted(); err != nil {
		return err
	}
	return nil
}

func (br *batcher) includeCompleted() error {
	for len(br.outstanding) > 0 {
		b := br.outstanding[br.nextId]
		if !b.succeeded {
			return nil
		}
		if err := br.updateDB(*b); err != nil {
			return err
		}
		status := br.su.Status()
		min := primitives.Slot(status.LowSlot)
		promote := br.outstanding[br.next(min, b.begin).id()]
		br.nextId = promote.id()
		delete(br.outstanding, b.id())
	}
	return nil
}

func (br *batcher) updateDB(b batch) error {
	return nil
}

func (br *batcher) next(min, upper primitives.Slot) batch {
	n := batch{begin: min}
	n.end = upper // Batches don't overlap because end is exclusive, begin is inclusive.
	if upper > br.size+min {
		n.begin = upper - br.size
	}

	return n
}

func newBatcher(size primitives.Slot, su *StatusUpdater, todo, done chan batch) *batcher {
	return &batcher{
		size: size,
		su:   su,
		todo: todo,
		done: done,
	}
}
