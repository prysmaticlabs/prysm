package backfill

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

var errSequencerMisconfigured = errors.New("backfill sequencer initialization error")
var errMaxBatches = errors.New("backfill batch requested in excess of max outstanding batches")
var errEndSequence = errors.New("sequence has terminated, no more backfill batches will be produced")

type batchSequencer struct {
	batcher batcher
	seq     []batch
}

/*
// Allocate a buffer the same size as the existing one.
shifted := make([]batch, len(c.seq))
// Fill the low end with the not importable batches, leaving the higher indices as the zero value.
copy(shifted, c.seq[i:])
// Offset into new list where the nil batches start.
nb := len(shifted) - i
// Handle edge case where list is totally drained by chaining off the last element and move nb forward.

	if nb == 0 {
		shifted[nb] = c.batcher.beforeBatch(importable[i])
		nb += 1
	}

// Populate newly opened elements in seq with the appropriate batch.

	for ; nb < len(shifted); nb++ {
		shifted[nb] = c.batcher.beforeBatch(shifted[nb-1])
	}

// Overwrite the existing sequence with the sequence containing outstanding batches and room for more.
c.seq = shifted
*/
func (c *batchSequencer) update(b batch) {
	done := 0
	for i := 0; i < len(c.seq); i++ {
		if c.seq[i].begin == b.begin {
			c.seq[i] = b
		}
		// Assumes invariant that batches complete and update is called in order.
		if c.seq[i].state == batchImportComplete {
			done += 1
			continue
		}
		// Move the unfinished batches to overwrite the finished ones.
		c.seq[i-done] = c.seq[i]
	}
	// Overwrite the moved batches with the next ones in the sequence.
	for i := len(c.seq) - done; i < len(c.seq); i++ {
		c.seq[i] = c.batcher.beforeBatch(c.seq[i-1])
	}
}

// TODO fix batchEndSequence circular invariant
func (c *batchSequencer) sequence() (batch, error) {
	// batch start slots are in descending order, c.seq[n].begin == c.seq[n+1].end
	for i := range c.seq {
		switch c.seq[i].state {
		case batchInit, batchErrRetryable:
			c.seq[i].state = batchSequenced
			return c.seq[i], nil
		case batchNil:
			if i == 0 {
				return batch{}, errSequencerMisconfigured
			}
			c.seq[i] = c.batcher.beforeBatch(c.seq[i-1])
			c.seq[i].state = batchSequenced
			return c.seq[i], nil
		case batchEndSequence:
			return batch{}, errors.Wrapf(errEndSequence, "LowSlot=%d", c.seq[i].begin)
		default:
			continue
		}
	}

	return batch{}, errMaxBatches
}

func (c *batchSequencer) importable() []batch {
	for i := range c.seq {
		if c.seq[i].state == batchImportable {
			continue
		}
		return c.seq[0:i]
	}
	return nil
}

func newBatchSequencer(nw int, min, max, size primitives.Slot) *batchSequencer {
	b := batcher{min: min, size: size}
	seq := make([]batch, nw)
	seq[0] = b.before(max)
	return &batchSequencer{batcher: b, seq: seq}
}

type batcher struct {
	min  primitives.Slot
	size primitives.Slot
}

func (r batcher) beforeBatch(upTo batch) batch {
	return r.before(upTo.begin)
}

func (r batcher) before(upTo primitives.Slot) batch {
	// upTo is an exclusive upper bound. Requesting a batch before the lower bound of backfill signals the end of the
	// backfill process.
	if upTo <= r.min {
		return batch{begin: upTo, end: upTo, state: batchEndSequence}
	}
	begin := r.min
	if upTo > r.size+r.min {
		begin = upTo - r.size
	}

	// batch.end is exclusive, .begin is inclusive, so the prev.end = next.begin
	return batch{begin: begin, end: upTo, state: batchInit}
}
