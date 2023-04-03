package sync

import (
	"context"
	"sort"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type canonicalChecker func(context.Context, [32]byte) (bool, error)

// filters all the provided blocks to ensure they are canonical
// and are strictly linear.
func filterCanonical(ctx context.Context, blks []blocks.ROBlock, prevRoot *[32]byte, canonical canonicalChecker) ([]blocks.ROBlock, []blocks.ROBlock, error) {
	seq := make([]blocks.ROBlock, 0, len(blks))
	nseq := make([]blocks.ROBlock, 0)
	for i, b := range blks {
		cb, err := canonical(ctx, b.Root())
		if err != nil {
			return nil, nil, err
		}
		if !cb {
			continue
		}
		// filterCanonical is called in batches, so prevRoot can be the last root from the previous batch.
		// prevRoot will be the zero value until we find the first canonical block in a given request.
		first := *prevRoot == [32]byte{}
		// We assume blocks are processed in order, so the previous canonical root should be the parent of the next.
		// If the current block isn't descended from the last, something is wrong. Append everything remaining
		// to the list of non-sequential blocks and stop building the canonical list.
		if !first && *prevRoot != b.Block().ParentRoot() {
			nseq = append(nseq, blks[i:]...)
			break
		}
		seq = append(seq, blks[i])
		// Set the previous root as the
		// newly added block's root
		currRoot := b.Root()
		*prevRoot = currRoot
	}
	return seq, nseq, nil
}

// returns a copy of the []ROBlock list in sorted order with duplicates removed
func sortedUniqueBlocks(blks []blocks.ROBlock) []blocks.ROBlock {
	// Remove duplicate blocks received
	sort.Sort(blocks.ROBlockSlice(blks))
	u := 0
	for i := 1; i < len(blks); i++ {
		if blks[i].Root() != blks[u].Root() {
			u += 1
			if u != i {
				blks[u] = blks[i]
			}
		}
	}
	return blks[:u+1]
}

type blockBatch struct {
	start  primitives.Slot
	end    primitives.Slot
	seq    []blocks.ROBlock
	nonseq []blocks.ROBlock
	err    error
}

func (bb blockBatch) RateLimitCost() int {
	return int(bb.end - bb.start)
}

func (bb blockBatch) Sequence() []blocks.ROBlock {
	return bb.seq
}

func (bb blockBatch) SequenceBroken() bool {
	return len(bb.nonseq) > 0
}

func (bb blockBatch) Err() error {
	return bb.err
}

type blockRangeBatcher struct {
	start       primitives.Slot
	end         primitives.Slot
	size        uint64
	db          db.NoHeadAccessDatabase
	limiter     *limiter
	isCanonical canonicalChecker
	ticker      *time.Ticker

	lastSeq [32]byte
	current *blockBatch
}

func (bb *blockRangeBatcher) genesisBlock(ctx context.Context) (blocks.ROBlock, error) {
	b, err := bb.db.GenesisBlock(ctx)
	if err != nil {
		return blocks.ROBlock{}, err
	}
	htr, err := b.Block().HashTreeRoot()
	if err != nil {
		return blocks.ROBlock{}, err
	}
	return blocks.NewROBlock(b, htr), nil
}

func newBlockBatch(start, reqEnd primitives.Slot, size uint64) (blockBatch, bool) {
	if start > reqEnd {
		return blockBatch{}, false
	}
	nb := blockBatch{start: start, end: start.Add(size - 1)}
	if nb.end > reqEnd {
		nb.end = reqEnd
	}
	return nb, true
}

func (bat blockBatch) Next(reqEnd primitives.Slot, size uint64) (blockBatch, bool) {
	if bat.SequenceBroken() {
		return blockBatch{}, false
	}
	return newBlockBatch(bat.end.Add(1), reqEnd, size)
}

func (bb *blockRangeBatcher) Next(ctx context.Context, stream libp2pcore.Stream) (blockBatch, bool) {
	var nb blockBatch
	var ok bool
	if bb.current != nil {
		current := *bb.current
		nb, ok = current.Next(bb.end, bb.size)
	} else {
		nb, ok = newBlockBatch(bb.start, bb.end, bb.size)
	}
	if !ok {
		return blockBatch{}, false
	}
	if err := bb.limiter.validateRequest(stream, bb.size); err != nil {
		return blockBatch{err: errors.Wrap(err, "throttled by rate limiter")}, false
	}

	// block if there is work to do, unless this is the first batch
	if bb.ticker != nil && bb.current != nil {
		<-bb.ticker.C
	}
	filter := filters.NewFilter().SetStartSlot(nb.start).SetEndSlot(nb.end)
	blks, roots, err := bb.db.Blocks(ctx, filter)
	if err != nil {
		return blockBatch{err: errors.Wrap(err, "Could not retrieve blocks")}, false
	}

	// make slice with extra +1 capacity in case we want to grow it to also hold the genesis block
	rob := make([]blocks.ROBlock, len(blks), len(blks)+1)
	/*
		goff := 0 // offset for genesis value
		if nb.start == 0 {
			gb, err := bb.genesisBlock(ctx)
			if err != nil {
				return blockBatch{err: errors.Wrap(err, "could not retrieve genesis block")}, false
			}
			rob = append(rob, blocks.ROBlock{}) // grow the slice to its capacity to hold the genesis block
			rob[0] = gb
			goff = 1
		}
		for i := 0; i < len(blks); i++ {
			rob[goff+i] = blocks.NewROBlock(blks[i], roots[i])
		}
	*/
	for i := 0; i < len(blks); i++ {
		rob[i] = blocks.NewROBlock(blks[i], roots[i])
	}

	// Filter and sort our retrieved blocks, so that
	// we only return valid sets of blocks.
	rob = sortedUniqueBlocks(rob)

	nb.seq, nb.nonseq, nb.err = filterCanonical(ctx, rob, &bb.lastSeq, bb.isCanonical)

	// Decrease allowed blocks capacity by the number of streamed blocks.
	bb.limiter.add(stream, int64(1+nb.end.SubSlot(nb.start)))
	bb.current = &nb
	return *bb.current, true
}
