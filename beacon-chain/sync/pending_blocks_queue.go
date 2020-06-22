package sync

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/exp/rand"
)

var processPendingBlocksPeriod = slotutil.DivideSlotBy(3 /* times per slot */)

// processes pending blocks queue on every processPendingBlocksPeriod
func (xx *Service) processPendingBlocksQueue() {
	ctx := context.Background()
	locker := new(sync.Mutex)
	runutil.RunEvery(xx.ctx, processPendingBlocksPeriod, func() {
		locker.Lock()
		if err := xx.processPendingBlocks(ctx); err != nil {
			log.WithError(err).Error("Failed to process pending blocks")
		}
		locker.Unlock()
	})
}

// processes the block tree inside the queue
func (xx *Service) processPendingBlocks(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingBlocks")
	defer span.End()

	pids := xx.p2p.Peers().Connected()
	if err := xx.validatePendingSlots(); err != nil {
		return errors.Wrap(err, "could not validate pending slots")
	}
	slots := xx.sortedPendingSlots()

	span.AddAttributes(
		trace.Int64Attribute("numSlots", int64(len(slots))),
		trace.Int64Attribute("numPeers", int64(len(pids))),
	)

	for _, slot := range slots {
		ctx, span := trace.StartSpan(ctx, "processPendingBlocks.InnerLoop")
		span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))

		xx.pendingQueueLock.RLock()
		b := xx.slotToPendingBlocks[slot]
		// Skip if block does not exist.
		if b == nil || b.Block == nil {
			xx.pendingQueueLock.RUnlock()
			span.End()
			continue
		}
		xx.pendingQueueLock.RUnlock()
		inPendingQueue := xx.seenPendingBlocks[bytesutil.ToBytes32(b.Block.ParentRoot)]

		inDB := xx.db.HasBlock(ctx, bytesutil.ToBytes32(b.Block.ParentRoot))
		hasPeer := len(pids) != 0

		// Only request for missing parent block if it's not in DB, not in pending cache
		// and has peer in the peer list.
		if !inPendingQueue && !inDB && hasPeer {
			log.WithFields(logrus.Fields{
				"currentSlot": b.Block.Slot,
				"parentRoot":  hex.EncodeToString(bytesutil.Trunc(b.Block.ParentRoot)),
			}).Info("Requesting parent block")
			req := [][]byte{b.Block.ParentRoot}

			// Start with a random peer to query, but choose the first peer in our unsorted list that claims to
			// have a head slot newer than the block slot we are requesting.
			pid := pids[rand.Int()%len(pids)]
			for _, p := range pids {
				cs, err := xx.p2p.Peers().ChainState(p)
				if err != nil {
					return errors.Wrap(err, "failed to read chain state for peer")
				}
				if cs != nil && cs.HeadSlot >= slot {
					pid = p
					break
				}
			}

			if err := xx.sendRecentBeaconBlocksRequest(ctx, req, pid); err != nil {
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

		blkRoot, err := stateutil.BlockRoot(b.Block)
		if err != nil {
			traceutil.AnnotateError(span, err)
			span.End()
			return err
		}

		if err := xx.chain.ReceiveBlockNoPubsub(ctx, b, blkRoot); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Block.Slot, err)
			traceutil.AnnotateError(span, err)
		}

		// Broadcasting the block again once a node is able to process it.
		if err := xx.p2p.Broadcast(ctx, b); err != nil {
			log.WithError(err).Error("Failed to broadcast block")
		}

		xx.pendingQueueLock.Lock()
		delete(xx.slotToPendingBlocks, slot)
		delete(xx.seenPendingBlocks, blkRoot)
		xx.pendingQueueLock.Unlock()

		log.WithFields(logrus.Fields{
			"slot":      slot,
			"blockRoot": hex.EncodeToString(bytesutil.Trunc(blkRoot[:])),
		}).Debug("Processed pending block and cleared it in cache")

		span.End()
	}

	return nil
}

func (xx *Service) sortedPendingSlots() []uint64 {
	xx.pendingQueueLock.RLock()
	defer xx.pendingQueueLock.RUnlock()

	slots := make([]uint64, 0, len(xx.slotToPendingBlocks))
	for slot := range xx.slotToPendingBlocks {
		slots = append(slots, slot)
	}
	sort.Slice(slots, func(i, j int) bool {
		return slots[i] < slots[j]
	})
	return slots
}

// validatePendingSlots validates the pending blocks
// by their slot. If they are before the current finalized
// checkpoint, these blocks are removed from the queue.
func (xx *Service) validatePendingSlots() error {
	xx.pendingQueueLock.Lock()
	defer xx.pendingQueueLock.Unlock()
	oldBlockRoots := make(map[[32]byte]bool)

	finalizedEpoch := xx.chain.FinalizedCheckpt().Epoch
	for s, b := range xx.slotToPendingBlocks {
		epoch := helpers.SlotToEpoch(s)
		// remove all descendant blocks of old blocks
		if oldBlockRoots[bytesutil.ToBytes32(b.Block.ParentRoot)] {
			root, err := stateutil.BlockRoot(b.Block)
			if err != nil {
				return err
			}
			oldBlockRoots[root] = true
			delete(xx.slotToPendingBlocks, s)
			delete(xx.seenPendingBlocks, root)
			continue
		}
		// don't process old blocks
		if finalizedEpoch > 0 && epoch <= finalizedEpoch {
			blkRoot, err := stateutil.BlockRoot(b.Block)
			if err != nil {
				return err
			}
			oldBlockRoots[blkRoot] = true
			delete(xx.slotToPendingBlocks, s)
			delete(xx.seenPendingBlocks, blkRoot)
		}
	}
	return nil
}

func (xx *Service) clearPendingSlots() {
	xx.pendingQueueLock.Lock()
	defer xx.pendingQueueLock.Unlock()
	xx.slotToPendingBlocks = make(map[uint64]*ethpb.SignedBeaconBlock)
	xx.seenPendingBlocks = make(map[[32]byte]bool)
}
