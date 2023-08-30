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

var errCannotDecreaseMinimum = errors.New("The minimum backfill slot can only be increased, not decreased")

// moveMinimum enables the backfill service to change the slot where the batcher will start replying with
// batch state batchEndSequence (signaling that no new batches will be produced). This is done in response to
// epochs advancing, which shrinks the gap between <checkpoint slot> and <current slot>-MIN_EPOCHS_FOR_BLOCK_REQUESTS,
// allowing the node to download a smaller number of blocks.
func (c *batchSequencer) moveMinimum(min primitives.Slot) error {
	if min < c.batcher.min {
		return errCannotDecreaseMinimum
	}
	c.batcher.min = min
	return nil
}

func (c *batchSequencer) minimum() primitives.Slot {
	return c.batcher.min
}

func (c *batchSequencer) update(b batch) {
	done := 0
	for i := 0; i < len(c.seq); i++ {
		if b.replaces(c.seq[i]) {
			c.seq[i] = b
		}
		// Assumes invariant that batches complete and update is called in order.
		// This should be true because the code using the sequencer doesn't know the expected parent
		// for a batch until it imports the previous batch.
		if c.seq[i].state == batchImportComplete {
			done += 1
			continue
		}
		// Move the unfinished batches to overwrite the finished ones.
		c.seq[i-done] = c.seq[i]
	}
	// Overwrite the moved batches with the next ones in the sequence.
	last := c.seq[len(c.seq)-1]
	for i := len(c.seq) - done; i < len(c.seq); i++ {
		c.seq[i] = c.batcher.beforeBatch(last)
		last = c.seq[i]
	}
}

func (c *batchSequencer) sequence() ([]batch, error) {
	s := make([]batch, 0)
	// batch start slots are in descending order, c.seq[n].begin == c.seq[n+1].end
	for i := range c.seq {
		switch c.seq[i].state {
		case batchInit, batchErrRetryable:
			c.seq[i].state = batchSequenced
			c.seq[i].inc()
			s = append(s, c.seq[i])
		case batchNil:
			if i == 0 {
				return nil, errSequencerMisconfigured
			}
			c.seq[i] = c.batcher.beforeBatch(c.seq[i-1])
			c.seq[i].state = batchSequenced
			c.seq[i].inc()
			s = append(s, c.seq[i])
		case batchEndSequence:
			if len(s) == 0 {
				s = append(s, c.seq[i])
			}
			break
		default:
			continue
		}
	}
	if len(s) == 0 {
		return nil, errMaxBatches
	}

	return s, nil
}

// TODO: write a method to answer to determine how many batches are left and how many are in progress
// look at .seq and check if there's an endseq in there.

func (c *batchSequencer) numTodo() int {
	if len(c.seq) == 0 {
		return 0
	}
	lowest := c.seq[len(c.seq)-1]
	todo := 0
	if lowest.state != batchEndSequence {
		todo = c.batcher.remaining(lowest.begin)
	}
	for _, b := range c.seq {
		switch b.state {
		case batchEndSequence, batchImportComplete, batchNil:
			continue
		default:
			todo += 1
		}
	}
	return todo
}

func (c *batchSequencer) importable() []batch {
	for i := range c.seq {
		if c.seq[i].state == batchImportable {
			continue
		}
		// as soon as we hit a batch with a different state, we return everything leading to it.
		// if the first element isn't importable, we'll return slice [0:0] aka nothing.
		return c.seq[0:i]
	}
	// if we hit this condition, it means every element had state = importable
	return c.seq
}

func newBatchSequencer(seqLen int, min, max, size primitives.Slot) *batchSequencer {
	b := batcher{min: min, size: size}
	seq := make([]batch, seqLen)
	seq[0] = b.before(max)
	return &batchSequencer{batcher: b, seq: seq}
}

type batcher struct {
	min  primitives.Slot
	size primitives.Slot
}

func (r batcher) remaining(upTo primitives.Slot) int {
	if r.min >= upTo {
		return 0
	}
	delta := upTo - r.min
	if delta%r.size != 0 {
		return int(delta/r.size) + 1
	}
	return int(delta / r.size)
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
