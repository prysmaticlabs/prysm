package stategen

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var ErrFutureSlotRequested = errors.New("cannot replay to future slots")
var ErrNoCanonicalBlockForSlot = errors.New("none of the blocks found in the db slot index are canonical")
var ErrNoBlocksBelowSlot = errors.New("no blocks found in db below slot")
var ErrInvalidDBBlock = errors.New("invalid block found in database")
var ErrGenesisBlockNotFound = errors.New("canonical genesis block not found in db")

// HistoryAccessor describes the minimum set of database methods needed to support the ReplayerBuilder
type HistoryAccessor interface {
	HighestSlotBlocksBelow(ctx context.Context, slot types.Slot) ([]block.SignedBeaconBlock, error)
	GenesisBlock(ctx context.Context) (block.SignedBeaconBlock, error)
	Block(ctx context.Context, blockRoot [32]byte) (block.SignedBeaconBlock, error)
	StateOrError(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
}

// CanonicalChecker is used by the Replayer to find the highest canonical block in the database <= the target slot.
// In practice this should be satisfied by a type that uses the fork choice store.
type CanonicalChecker interface {
	IsCanonical(ctx context.Context, blockRoot [32]byte) (bool, error)
}

// CurrentSlotter is used to ensure requestedSlot < currentSlot, to fail fast on requests to replay to future slots
type CurrentSlotter interface {
	CurrentSlot() types.Slot
}

// Replayer encapsulates database query and replay logic. It can be constructed via a StateReplayerBuilder
type Replayer interface {
	// ReplayBlocks replays the blocks the Replayer knows about based on Builder params
	ReplayBlocks(ctx context.Context) (state.BeaconState, error)
	// ReplayToSlot invokes ReplayBlocks under the hood,
	// but then also runs process_slots to advance the state past the root or slot used in the builder.
	// For example, if you wanted the state to be at the target slot, but only integrating blocks up to
	// slot-1, you could request Builder.ForSlot(slot-1).ReplayToSlot(slot)
	ReplayToSlot(ctx context.Context, target types.Slot) (state.BeaconState, error)
}

var _ Replayer = &stateReplayer{}

type stateReplayer struct {
	s           state.BeaconState
	descendants []block.SignedBeaconBlock
	target      types.Slot
	method      retrievalMethod
	chainer     chainer
}

// ReplayBlocks applies all the blocks that were accumulated when building the Replayer
// this method relies on the correctness of the code that constructed the Replayer data
func (rs *stateReplayer) ReplayBlocks(ctx context.Context) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.stateReplayer.ReplayBlocks")
	defer span.End()

	var s state.BeaconState
	var descendants []block.SignedBeaconBlock
	var err error
	switch rs.method {
	case forSlot:
		s, descendants, err = rs.chainer.chainForSlot(ctx, rs.target)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("Replayer initialized using unknown state retrieval method")
	}

	start := time.Now()
	log.WithFields(logrus.Fields{
		"startSlot": s.Slot(),
		"endSlot":   rs.target,
		"diff":      rs.target - s.Slot(),
	}).Debug("replaying canonical blocks from most recent state")

	for _, b := range descendants {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		s, err = executeStateTransitionStateGen(ctx, s, b)
		if err != nil {
			return nil, err
		}
	}
	if rs.target > s.Slot() {
		s, err = ReplayProcessSlots(ctx, s, rs.target)
		if err != nil {
			return nil, err
		}
	}

	duration := time.Since(start)
	log.WithFields(logrus.Fields{
		"duration": duration,
	}).Debug("finished calling process_blocks on all blocks in ReplayBlocks")
	return s, nil
}

// ReplayToSlot invokes ReplayBlocks under the hood,
// but then also runs process_slots to advance the state past the root or slot used in the builder.
// for example, if you wanted the state to be at the target slot, but only integrating blocks up to
// slot-1, you could request Builder.ForSlot(slot-1).ReplayToSlot(slot)
func (rs *stateReplayer) ReplayToSlot(ctx context.Context, target types.Slot) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "stateGen.stateReplayer.ReplayToSlot")
	defer span.End()

	s, err := rs.ReplayBlocks(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReplayBlocks")
	}

	if target > s.Slot() {
		start := time.Now()
		log.WithFields(logrus.Fields{
			"startSlot": s.Slot(),
			"endSlot":   target,
			"diff":      target - s.Slot(),
		}).Debug("calling process_slots on remaining slots")

		if rs.target > s.Slot() {
			// err will be handled after the bookend log
			s, err = ReplayProcessSlots(ctx, s, target)
		}

		duration := time.Since(start)
		log.WithFields(logrus.Fields{
			"duration": duration,
		}).Debug("time spent in process_slots")
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("ReplayToSlot failed to seek to slot %d after applying blocks", target))
		}
	}
	return s, nil
}

// ReplayerBuilder creates a Replayer that can be used to obtain a state at a specified slot or root
// (only ForSlot implemented so far).
// See documentation on Replayer for more on how to use this to obtain pre/post-block states
type ReplayerBuilder interface {
	// ForSlot creates a builder that will create a state that includes blocks up to and including the requested slot
	// The resulting Replayer will always yield a state with .Slot=target; if there are skipped blocks
	// between the highest canonical block in the db and the target, the replayer will fast-forward past the intervening
	// slots via process_slots.
	ForSlot(target types.Slot) Replayer
}

// NewCanonicalBuilder handles initializing the default concrete ReplayerBuilder implementation.
func NewCanonicalBuilder(h HistoryAccessor, c CanonicalChecker, cs CurrentSlotter) *canonicalBuilder {
	return &canonicalBuilder{
		chainer: &canonicalChainer{
			h:  h,
			c:  c,
			cs: cs,
		},
	}
}

type retrievalMethod int

const (
	forSlot retrievalMethod = iota
)

// canonicalBuilder builds a Replayer that uses a combination of database queries and a
// CanonicalChecker (which should usually be a fork choice store implementing an IsCanonical method)
// to determine the canonical chain and apply it to generate the desired state.
type canonicalBuilder struct {
	chainer chainer
}

var _ ReplayerBuilder = &canonicalBuilder{}

func (r *canonicalBuilder) ForSlot(target types.Slot) Replayer {
	return &stateReplayer{chainer: r.chainer, method: forSlot, target: target}
}

// chainer is responsible for supplying the chain components necessary to rebuild a state,
// namely a starting BeaconState and all available blocks from the starting state up to and including the target slot
type chainer interface {
	chainForSlot(ctx context.Context, target types.Slot) (state.BeaconState, []block.SignedBeaconBlock, error)
}

type canonicalChainer struct {
	h  HistoryAccessor
	c  CanonicalChecker
	cs CurrentSlotter
}

var _ chainer = &canonicalChainer{}

// ChainForSlot creates a value that satisfies the Replayer interface via db queries
// and the stategen transition helper methods. This implementation uses the following algorithm:
// - find the highest canonical block >= the target slot
// - starting with this block, recursively search backwards for a stored state, and accumulate intervening blocks
func (r *canonicalChainer) chainForSlot(ctx context.Context, target types.Slot) (state.BeaconState, []block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "canonicalChainer.ForSlot")
	defer span.End()
	currentSlot := r.cs.CurrentSlot()
	if target > currentSlot {
		return nil, nil, errors.Wrap(ErrFutureSlotRequested, fmt.Sprintf("requested=%d, current=%d", target, currentSlot))
	}
	_, b, err := r.canonicalBlockForSlot(ctx, target)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("unable to find replay data for slot=%d", target))
	}
	s, descendants, err := r.ancestorChain(ctx, b)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query for ancestor and descendant blocks")
	}

	return s, descendants, nil
}

// canonicalBlockForSlot uses HighestSlotBlocksBelow(target+1) and the CanonicalChecker
// to find the highest canonical block available to replay to the given slot.
func (r *canonicalChainer) canonicalBlockForSlot(ctx context.Context, target types.Slot) ([32]byte, block.SignedBeaconBlock, error) {
	for target > 0 {
		hbs, err := r.h.HighestSlotBlocksBelow(ctx, target+1)
		if err != nil {
			return [32]byte{}, nil, errors.Wrap(err, fmt.Sprintf("error finding highest block w/ slot <= %d", target))
		}
		if len(hbs) == 0 {
			return [32]byte{}, nil, errors.Wrap(ErrNoBlocksBelowSlot, fmt.Sprintf("slot=%d", target))
		}
		r, b, err := r.bestForSlot(ctx, hbs)
		if err == nil {
			// we found a valid, canonical block!
			return r, b, nil
		}

		// we found a block, but it wasn't considered canonical - keep looking
		if errors.Is(err, ErrNoCanonicalBlockForSlot) {
			// break once we've seen slot 0 (and prevent underflow)
			if hbs[0].Block().Slot() == 0 {
				break
			}
			target = hbs[0].Block().Slot() - 1
			continue
		}
		return [32]byte{}, nil, err
	}
	b, err := r.h.GenesisBlock(ctx)
	if err != nil {
		return [32]byte{}, nil, errors.Wrap(err, "db error while retrieving genesis block")
	}
	root, _, err := r.bestForSlot(ctx, []block.SignedBeaconBlock{b})
	if err != nil {
		return [32]byte{}, nil, errors.Wrap(err, "problem retrieving genesis block")
	}
	return root, b, nil
}

// bestForSlot encapsulates several messy realities of the underlying db code, looping through multiple blocks,
// performing null/validity checks, and using CanonicalChecker to only pick canonical blocks.
func (r *canonicalChainer) bestForSlot(ctx context.Context, hbs []block.SignedBeaconBlock) ([32]byte, block.SignedBeaconBlock, error) {
	for _, b := range hbs {
		if helpers.BeaconBlockIsNil(b) != nil {
			continue
		}
		root, err := b.Block().HashTreeRoot()
		if err != nil {
			// use this error message to wrap a sentinel error for error type matching
			wrapped := errors.Wrap(ErrInvalidDBBlock, err.Error())
			msg := fmt.Sprintf("could not compute hash_tree_root for block @ slot=%d", b.Block().Slot())
			return [32]byte{}, nil, errors.Wrap(wrapped, msg)
		}
		canon, err := r.c.IsCanonical(ctx, root)
		if err != nil {
			return [32]byte{}, nil, errors.Wrap(err, "error from blockchain info fetcher IsCanonical check")
		}
		if canon {
			return root, b, nil
		}
	}
	return [32]byte{}, nil, errors.Wrap(ErrNoCanonicalBlockForSlot, "no good block for slot")
}

// ancestorChain works backwards through the chain lineage, accumulating blocks and checking for a saved state
// if it finds a saved state the the tail block was descended from, it returns this state and
// all blocks in the lineage, including the tail block. blocks are returned in ascending order.
// note that this function assumes that the tail is a canonical block, and therefore assumes that
// all ancestors are also canonical.
func (r *canonicalChainer) ancestorChain(ctx context.Context, tail block.SignedBeaconBlock) (state.BeaconState, []block.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "canonicalChainer.ancestorChain")
	defer span.End()
	chain := make([]block.SignedBeaconBlock, 0)
	for {
		if err := ctx.Err(); err != nil {
			msg := fmt.Sprintf("context canceled while finding ancestors of block @ slot %d", tail.Block().Slot())
			return nil, nil, errors.Wrap(err, msg)
		}
		b := tail.Block()
		// compute hash_tree_root of current block and try to look up the corresponding state
		root, err := b.HashTreeRoot()
		if err != nil {
			msg := fmt.Sprintf("could not compute htr for descendant block @ slot=%d", b.Slot())
			return nil, nil, errors.Wrap(err, msg)
		}
		st, err := r.h.StateOrError(ctx, root)
		// err from StateOrError == nil, we've got a real state - the job is done!
		if err == nil {
			// this handles cases where we find the state right away. return empty list of blocks
			// we found the state by the root of the head, meaning it has already been applied.
			// we only want to return the blocks descended from it.
			reverseChain(chain)
			return st, chain, nil
		}
		// ErrNotFoundState errors are fine, but other errors mean something is wrong with the db
		if !errors.Is(err, db.ErrNotFoundState) {
			return nil, nil, errors.Wrap(err, fmt.Sprintf("error querying database for state w/ block root = %#x", root))
		}
		parent, err := r.h.Block(ctx, bytesutil.ToBytes32(b.ParentRoot()))
		if err != nil {
			msg := fmt.Sprintf("db error when retrieving parent of block @ slot=%d by root=%#x", b.Slot(), b.ParentRoot())
			return nil, nil, errors.Wrap(err, msg)
		}
		if helpers.BeaconBlockIsNil(parent) != nil {
			msg := fmt.Sprintf("unable to retrieve parent of block @ slot=%d by root=%#x", b.Slot(), b.ParentRoot())
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
