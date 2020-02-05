package sync

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/exp/rand"
)

var processPendingBlocksPeriod = time.Duration(params.BeaconConfig().SecondsPerSlot/3) * time.Second

// processes pending blocks queue on every processPendingBlocksPeriod
func (r *Service) processPendingBlocksQueue() {
	ctx := context.Background()
	locker := new(sync.Mutex)
	runutil.RunEvery(r.ctx, processPendingBlocksPeriod, func() {
		locker.Lock()
		r.processPendingBlocks(ctx)
		locker.Unlock()
	})
}

// processes the block tree inside the queue
func (r *Service) processPendingBlocks(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingBlocks")
	defer span.End()

	pids := r.p2p.Peers().Connected()
	if err := r.validatePendingSlots(); err != nil {
		return errors.Wrap(err, "could not validate pending slots")
	}
	slots := r.sortedPendingSlots()

	span.AddAttributes(
		trace.Int64Attribute("numSlots", int64(len(slots))),
		trace.Int64Attribute("numPeers", int64(len(pids))),
	)

	for _, s := range slots {
		ctx, span := trace.StartSpan(ctx, "processPendingBlocks.InnerLoop")
		span.AddAttributes(trace.Int64Attribute("slot", int64(s)))

		r.pendingQueueLock.RLock()
		b := r.slotToPendingBlocks[uint64(s)]
		// Skip if block does not exist.
		if b == nil || b.Block == nil {
			r.pendingQueueLock.RUnlock()
			span.End()
			continue
		}
		r.pendingQueueLock.RUnlock()
		inPendingQueue := r.seenPendingBlocks[bytesutil.ToBytes32(b.Block.ParentRoot)]

		inDB := r.db.HasBlock(ctx, bytesutil.ToBytes32(b.Block.ParentRoot))
		hasPeer := len(pids) != 0

		// Only request for missing parent block if it's not in DB, not in pending cache
		// and has peer in the peer list.
		if !inPendingQueue && !inDB && hasPeer {
			log.WithFields(logrus.Fields{
				"currentSlot": b.Block.Slot,
				"parentRoot":  hex.EncodeToString(bytesutil.Trunc(b.Block.ParentRoot)),
			}).Info("Requesting parent block")
			req := [][32]byte{bytesutil.ToBytes32(b.Block.ParentRoot)}

			// Start with a random peer to query, but choose the first peer in our unsorted list that claims to
			// have a head slot newer than the block slot we are requesting.
			pid := pids[rand.Int()%len(pids)]
			for _, p := range pids {
				if cs, _ := r.p2p.Peers().ChainState(p); cs != nil && cs.HeadSlot >= uint64(s) {
					pid = p
					break
				}
			}

			if err := r.sendRecentBeaconBlocksRequest(ctx, req, pid); err != nil {
				traceutil.AnnotateError(span, err)
				log.Errorf("Could not send recent block request: %v", err)
			}
			span.End()
			continue
		}

		if !inDB {
			span.End()
			continue
		}

		if err := r.chain.ReceiveBlockNoPubsub(ctx, b); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Block.Slot, err)
			traceutil.AnnotateError(span, err)
		}

		// Broadcasting the block again once a node is able to process it.
		if err := r.p2p.Broadcast(ctx, b); err != nil {
			log.WithError(err).Error("Failed to broadcast block")
		}

		blkRoot, err := ssz.HashTreeRoot(b.Block)
		if err != nil {
			traceutil.AnnotateError(span, err)
			span.End()
			return err
		}

		r.pendingQueueLock.Lock()
		delete(r.slotToPendingBlocks, uint64(s))
		delete(r.seenPendingBlocks, blkRoot)
		r.pendingQueueLock.Unlock()

		log.WithFields(logrus.Fields{
			"slot":      s,
			"blockRoot": hex.EncodeToString(bytesutil.Trunc(blkRoot[:])),
		}).Info("Processed pending block and cleared it in cache")

		span.End()
	}

	return nil
}

func (r *Service) sortedPendingSlots() []int {
	r.pendingQueueLock.RLock()
	defer r.pendingQueueLock.RUnlock()

	slots := make([]int, 0, len(r.slotToPendingBlocks))
	for s := range r.slotToPendingBlocks {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)
	return slots
}

// validatePendingSlots validates the pending blocks
// by their slot. If they are before the current finalized
// checkpoint, these blocks are removed from the queue.
func (r *Service) validatePendingSlots() error {
	r.pendingQueueLock.Lock()
	defer r.pendingQueueLock.Unlock()
	oldBlockRoots := make(map[[32]byte]bool)

	finalizedEpoch := r.chain.FinalizedCheckpt().Epoch
	for s, b := range r.slotToPendingBlocks {
		epoch := helpers.SlotToEpoch(s)
		// remove all descendant blocks of old blocks
		if oldBlockRoots[bytesutil.ToBytes32(b.Block.ParentRoot)] {
			root, err := ssz.HashTreeRoot(b.Block)
			if err != nil {
				return err
			}
			oldBlockRoots[root] = true
			delete(r.slotToPendingBlocks, s)
			delete(r.seenPendingBlocks, root)
			continue
		}
		// don't process old blocks
		if finalizedEpoch > 0 && epoch <= finalizedEpoch {
			blkRoot, err := ssz.HashTreeRoot(b.Block)
			if err != nil {
				return err
			}
			oldBlockRoots[blkRoot] = true
			delete(r.slotToPendingBlocks, s)
			delete(r.seenPendingBlocks, blkRoot)
		}
	}
	return nil
}

func (r *Service) clearPendingSlots() {
	r.pendingQueueLock.Lock()
	defer r.pendingQueueLock.Unlock()
	r.slotToPendingBlocks = make(map[uint64]*ethpb.SignedBeaconBlock)
	r.seenPendingBlocks = make(map[[32]byte]bool)
}
