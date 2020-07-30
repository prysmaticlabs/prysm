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
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var processPendingBlocksPeriod = slotutil.DivideSlotBy(3 /* times per slot */)

// processes pending blocks queue on every processPendingBlocksPeriod
func (s *Service) processPendingBlocksQueue() {
	ctx := context.Background()
	// Prevents multiple queue processing goroutines (invoked by RunEvery) from contending for data.
	locker := new(sync.Mutex)
	runutil.RunEvery(s.ctx, processPendingBlocksPeriod, func() {
		locker.Lock()
		if err := s.processPendingBlocks(ctx); err != nil {
			log.WithError(err).Error("Failed to process pending blocks")
		}
		locker.Unlock()
	})
}

// processes the block tree inside the queue
func (s *Service) processPendingBlocks(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingBlocks")
	defer span.End()

	pids := s.p2p.Peers().Connected()
	if err := s.validatePendingSlots(); err != nil {
		return errors.Wrap(err, "could not validate pending slots")
	}
	slots := s.sortedPendingSlots()

	span.AddAttributes(
		trace.Int64Attribute("numSlots", int64(len(slots))),
		trace.Int64Attribute("numPeers", int64(len(pids))),
	)

	randGen := rand.NewGenerator()
	for _, slot := range slots {
		ctx, span := trace.StartSpan(ctx, "processPendingBlocks.InnerLoop")
		span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))

		s.pendingQueueLock.RLock()
		b := s.slotToPendingBlocks[slot]
		// Skip if block does not exist.
		if b == nil || b.Block == nil {
			s.pendingQueueLock.RUnlock()
			span.End()
			continue
		}
		inPendingQueue := s.seenPendingBlocks[bytesutil.ToBytes32(b.Block.ParentRoot)]
		s.pendingQueueLock.RUnlock()

		blkRoot, err := stateutil.BlockRoot(b.Block)
		if err != nil {
			traceutil.AnnotateError(span, err)
			span.End()
			return err
		}
		parentIsBad := s.hasBadBlock(bytesutil.ToBytes32(b.Block.ParentRoot))
		blockIsBad := s.hasBadBlock(blkRoot)
		// Check if parent is a bad block.
		if parentIsBad || blockIsBad {
			// Set block as bad if its parent block is bad too.
			if parentIsBad {
				s.setBadBlock(blkRoot)
			}
			// Remove block from queue.
			s.pendingQueueLock.Lock()
			delete(s.slotToPendingBlocks, slot)
			delete(s.seenPendingBlocks, blkRoot)
			s.pendingQueueLock.Unlock()
			span.End()
			continue
		}

		inDB := s.db.HasBlock(ctx, bytesutil.ToBytes32(b.Block.ParentRoot))
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
			pid := pids[randGen.Int()%len(pids)]
			for _, p := range pids {
				cs, err := s.p2p.Peers().ChainState(p)
				if err != nil {
					return errors.Wrap(err, "failed to read chain state for peer")
				}
				if cs != nil && cs.HeadSlot >= slot {
					pid = p
					break
				}
			}

			if err := s.sendRecentBeaconBlocksRequest(ctx, req, pid); err != nil {
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

		if err := s.chain.ReceiveBlock(ctx, b, blkRoot); err != nil {
			log.Errorf("Could not process block from slot %d: %v", b.Block.Slot, err)
			s.setBadBlock(blkRoot)
			traceutil.AnnotateError(span, err)
		}

		// Broadcasting the block again once a node is able to process it.
		if err := s.p2p.Broadcast(ctx, b); err != nil {
			log.WithError(err).Error("Failed to broadcast block")
		}

		s.pendingQueueLock.Lock()
		delete(s.slotToPendingBlocks, slot)
		delete(s.seenPendingBlocks, blkRoot)
		s.pendingQueueLock.Unlock()

		log.WithFields(logrus.Fields{
			"slot":      slot,
			"blockRoot": hex.EncodeToString(bytesutil.Trunc(blkRoot[:])),
		}).Debug("Processed pending block and cleared it in cache")

		span.End()
	}

	return nil
}

func (s *Service) sortedPendingSlots() []uint64 {
	s.pendingQueueLock.RLock()
	defer s.pendingQueueLock.RUnlock()

	slots := make([]uint64, 0, len(s.slotToPendingBlocks))
	for slot := range s.slotToPendingBlocks {
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
func (s *Service) validatePendingSlots() error {
	s.pendingQueueLock.Lock()
	defer s.pendingQueueLock.Unlock()
	oldBlockRoots := make(map[[32]byte]bool)

	finalizedEpoch := s.chain.FinalizedCheckpt().Epoch
	for slot, b := range s.slotToPendingBlocks {
		epoch := helpers.SlotToEpoch(slot)
		// remove all descendant blocks of old blocks
		if oldBlockRoots[bytesutil.ToBytes32(b.Block.ParentRoot)] {
			root, err := stateutil.BlockRoot(b.Block)
			if err != nil {
				return err
			}
			oldBlockRoots[root] = true
			delete(s.slotToPendingBlocks, slot)
			delete(s.seenPendingBlocks, root)
			continue
		}
		// don't process old blocks
		if finalizedEpoch > 0 && epoch <= finalizedEpoch {
			blkRoot, err := stateutil.BlockRoot(b.Block)
			if err != nil {
				return err
			}
			oldBlockRoots[blkRoot] = true
			delete(s.slotToPendingBlocks, slot)
			delete(s.seenPendingBlocks, blkRoot)
		}
	}
	return nil
}

func (s *Service) clearPendingSlots() {
	s.pendingQueueLock.Lock()
	defer s.pendingQueueLock.Unlock()
	s.slotToPendingBlocks = make(map[uint64]*ethpb.SignedBeaconBlock)
	s.seenPendingBlocks = make(map[[32]byte]bool)
}
