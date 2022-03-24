package stategen

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"go.opencensus.io/trace"
)

func WithCache(c CachedGetter) CanonicalHistoryOption {
	return func(h *CanonicalHistory) {
		h.cache = c
	}
}

type CanonicalHistoryOption func(*CanonicalHistory)

func NewCanonicalHistory(h HistoryAccessor, cc CanonicalChecker, cs CurrentSlotter, opts ...CanonicalHistoryOption) *CanonicalHistory {
	ch := &CanonicalHistory{
		h:  h,
		cc: cc,
		cs: cs,
	}
	for _, o := range opts {
		o(ch)
	}
	return ch
}

type CanonicalHistory struct {
	h     HistoryAccessor
	cc    CanonicalChecker
	cs    CurrentSlotter
	cache CachedGetter
}

func (ch *CanonicalHistory) ReplayerForSlot(target types.Slot) Replayer {
	return &stateReplayer{chainer: ch, method: forSlot, target: target}
}

func (c *CanonicalHistory) BlockForSlot(ctx context.Context, target types.Slot) ([32]byte, block.SignedBeaconBlock, error) {
	currentSlot := c.cs.CurrentSlot()
	if target > currentSlot {
		return [32]byte{}, nil, errors.Wrap(ErrFutureSlotRequested, fmt.Sprintf("requested=%d, current=%d", target, currentSlot))
	}
	for target > 0 {
		if ctx.Err() != nil {
			return [32]byte{}, nil, errors.Wrap(ctx.Err(), "context canceled during canonicalBlockForSlot")
		}
		hbs, err := c.h.HighestSlotBlocksBelow(ctx, target+1)
		if err != nil {
			return [32]byte{}, nil, errors.Wrap(err, fmt.Sprintf("error finding highest block w/ slot <= %d", target))
		}
		if len(hbs) == 0 {
			return [32]byte{}, nil, errors.Wrap(ErrNoBlocksBelowSlot, fmt.Sprintf("slot=%d", target))
		}
		r, b, err := c.bestForSlot(ctx, hbs)
		if err == nil {
			// we found a valid, canonical block!
			return r, b, nil
		}

		// we found a block, but it wasn't considered canonical - keep looking
		if errors.Is(err, ErrNoCanonicalBlockForSlot) {
			// break once we've seen slot 0 (and prevent underflow)
			if hbs[0].Block().Slot() == params.BeaconConfig().GenesisSlot {
				break
			}
			target = hbs[0].Block().Slot() - 1
			continue
		}
		return [32]byte{}, nil, err
	}
	b, err := c.h.GenesisBlock(ctx)
	if err != nil {
		return [32]byte{}, nil, errors.Wrap(err, "db error while retrieving genesis block")
	}
	root, _, err := c.bestForSlot(ctx, []block.SignedBeaconBlock{b})
	if err != nil {
		return [32]byte{}, nil, errors.Wrap(err, "problem retrieving genesis block")
	}
	return root, b, nil
}

// bestForSlot encapsulates several messy realities of the underlying db code, looping through multiple blocks,
// performing null/validity checks, and using CanonicalChecker to only pick canonical blocks.
func (c *CanonicalHistory) bestForSlot(ctx context.Context, hbs []block.SignedBeaconBlock) ([32]byte, block.SignedBeaconBlock, error) {
	for _, b := range hbs {
		if helpers.BeaconBlockIsNil(b) != nil {
			continue
		}
		root, err := b.Block().HashTreeRoot()
		if err != nil {
			// use this error message to wrap a sentinel error for error type matching
			wrapped := errors.Wrap(ErrInvalidDBBlock, err.Error())
			msg := fmt.Sprintf("could not compute hash_tree_root for block at slot=%d", b.Block().Slot())
			return [32]byte{}, nil, errors.Wrap(wrapped, msg)
		}
		canon, err := c.cc.IsCanonical(ctx, root)
		if err != nil {
			return [32]byte{}, nil, errors.Wrap(err, "replayer could not check if block is canonical")
		}
		if canon {
			return root, b, nil
		}
	}
	return [32]byte{}, nil, errors.Wrap(ErrNoCanonicalBlockForSlot, "no good block for slot")
}

// ChainForSlot creates a value that satisfies the Replayer interface via db queries
// and the stategen transition helper methods. This implementation uses the following algorithm:
// - find the highest canonical block <= the target slot
// - starting with this block, recursively search backwards for a stored state, and accumulate intervening blocks
func (c *CanonicalHistory) chainForSlot(ctx context.Context, target types.Slot) (state.BeaconState, []block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "canonicalChainer.chainForSlot")
	defer span.End()
	_, b, err := c.BlockForSlot(ctx, target)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("unable to find replay data for slot=%d", target))
	}
	s, descendants, err := c.ancestorChain(ctx, b)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query for ancestor and descendant blocks")
	}

	return s, descendants, nil
}

func (c *CanonicalHistory) getState(ctx context.Context, root [32]byte) (state.BeaconState, error) {
	if c.cache != nil {
		st, err := c.cache.ByRoot(root)
		if err == nil {
			return st, nil
		}
		if !errors.Is(err, ErrNotInCache) {
			return nil, errors.Wrap(err, "error reading from state cache during state replay")
		}
	}
	return c.h.StateOrError(ctx, root)
}

// ancestorChain works backwards through the chain lineage, accumulating blocks and checking for a saved state.
// If it finds a saved state that the tail block was descended from, it returns this state and
// all blocks in the lineage, including the tail block. Blocks are returned in ascending order.
// Note that this function assumes that the tail is a canonical block, and therefore assumes that
// all ancestors are also canonical.
func (c *CanonicalHistory) ancestorChain(ctx context.Context, tail block.SignedBeaconBlock) (state.BeaconState, []block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "canonicalChainer.ancestorChain")
	defer span.End()
	chain := make([]block.SignedBeaconBlock, 0)
	for {
		if err := ctx.Err(); err != nil {
			msg := fmt.Sprintf("context canceled while finding ancestors of block at slot %d", tail.Block().Slot())
			return nil, nil, errors.Wrap(err, msg)
		}
		b := tail.Block()
		// compute hash_tree_root of current block and try to look up the corresponding state
		root, err := b.HashTreeRoot()
		if err != nil {
			msg := fmt.Sprintf("could not compute htr for descendant block at slot=%d", b.Slot())
			return nil, nil, errors.Wrap(err, msg)
		}
		st, err := c.getState(ctx, root)
		// err == nil, we've got a real state - the job is done!
		// Note: in cases where there are skipped slots we could find a state that is a descendant
		// of the block we are searching for. We don't want to return a future block, so in this case
		// we keep working backwards.
		if err == nil && st.Slot() == b.Slot() {
			// we found the state by the root of the head, meaning it has already been applied.
			// we only want to return the blocks descended from it.
			reverseChain(chain)
			return st, chain, nil
		}
		// ErrNotFoundState errors are fine, but other errors mean something is wrong with the db
		if err != nil && !errors.Is(err, db.ErrNotFoundState) {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("error querying database for state w/ block root = %#x", root))
		}
		parent, err := c.h.Block(ctx, bytesutil.ToBytes32(b.ParentRoot()))
		if err != nil {
			msg := fmt.Sprintf("db error when retrieving parent of block at slot=%d by root=%#x", b.Slot(), b.ParentRoot())
			return nil, nil, errors.Wrap(err, msg)
		}
		if helpers.BeaconBlockIsNil(parent) != nil {
			msg := fmt.Sprintf("unable to retrieve parent of block at slot=%d by root=%#x", b.Slot(), b.ParentRoot())
			return nil, nil, errors.Wrap(db.ErrNotFound, msg)
		}
		chain = append(chain, tail)
		tail = parent
	}
}

func reverseChain(c []block.SignedBeaconBlock) {
	last := len(c) - 1
	swaps := (last + 1) / 2
	for i := 0; i < swaps; i++ {
		c[i], c[last-i] = c[last-i], c[i]
	}
}
