package sync

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var processPendingBlocksPeriod = slotutil.DivideSlotBy(3 /* times per slot */)

const maxPeerRequest = 50
const numOfTries = 5

// processes pending blocks queue on every processPendingBlocksPeriod
func (s *Service) processPendingBlocksQueue() {
	// Prevents multiple queue processing goroutines (invoked by RunEvery) from contending for data.
	locker := new(sync.Mutex)
	runutil.RunEvery(s.ctx, processPendingBlocksPeriod, func() {
		locker.Lock()
		if err := s.processPendingBlocks(s.ctx); err != nil {
			log.WithError(err).Debug("Failed to process pending blocks")
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
	var parentRoots [][32]byte

	span.AddAttributes(
		trace.Int64Attribute("numSlots", int64(len(slots))),
		trace.Int64Attribute("numPeers", int64(len(pids))),
	)

	randGen := rand.NewGenerator()
	for _, slot := range slots {
		ctx, span := trace.StartSpan(ctx, "processPendingBlocks.InnerLoop")
		span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))

		s.pendingQueueLock.RLock()
		bs := s.slotToPendingBlocks[slot]
		// Skip if there's no block in the queue.
		if len(bs) == 0 {
			s.pendingQueueLock.RUnlock()
			span.End()
			continue
		}
		s.pendingQueueLock.RUnlock()

		// Loop through the pending queue and mark the potential parent blocks as seen.
		for _, b := range bs {
			if b == nil || b.Block == nil {
				span.End()
				continue
			}

			s.pendingQueueLock.RLock()
			inPendingQueue := s.seenPendingBlocks[bytesutil.ToBytes32(b.Block.ParentRoot)]
			s.pendingQueueLock.RUnlock()

			blkRoot, err := b.Block.HashTreeRoot()
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
					s.setBadBlock(ctx, blkRoot)
				}
				// Remove block from queue.
				s.pendingQueueLock.Lock()
				s.deleteBlockFromPendingQueue(slot, b, blkRoot)
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
				}).Debug("Requesting parent block")
				parentRoots = append(parentRoots, bytesutil.ToBytes32(b.Block.ParentRoot))

				span.End()
				continue
			}

			if !inDB {
				span.End()
				continue
			}
			if err := s.chain.ReceiveBlock(ctx, b, blkRoot); err != nil {
				log.Debugf("Could not process block from slot %d: %v", b.Block.Slot, err)
				s.setBadBlock(ctx, blkRoot)
				traceutil.AnnotateError(span, err)
			}

			// Broadcasting the block again once a node is able to process it.
			if err := s.p2p.Broadcast(ctx, b); err != nil {
				log.WithError(err).Debug("Failed to broadcast block")
			}

			s.pendingQueueLock.Lock()
			s.deleteBlockFromPendingQueue(slot, b, blkRoot)
			s.pendingQueueLock.Unlock()

			log.WithFields(logrus.Fields{
				"slot":      slot,
				"blockRoot": hex.EncodeToString(bytesutil.Trunc(blkRoot[:])),
			}).Debug("Processed pending block and cleared it in cache")

			span.End()
		}
	}

	return s.sendBatchRootRequest(ctx, parentRoots, randGen)
}

func (s *Service) sendBatchRootRequest(ctx context.Context, roots [][32]byte, randGen *rand.Rand) error {
	ctx, span := trace.StartSpan(ctx, "sendBatchRootRequest")
	defer span.End()

	if len(roots) == 0 {
		return nil
	}

	_, bestPeers := s.p2p.Peers().BestFinalized(maxPeerRequest, s.chain.FinalizedCheckpt().Epoch)
	if len(bestPeers) == 0 {
		return nil
	}
	roots = s.dedupRoots(roots)
	// Randomly choose a peer to query from our best peers. If that peer cannot return
	// all the requested blocks, we randomly select another peer.
	pid := bestPeers[randGen.Int()%len(bestPeers)]
	for i := 0; i < numOfTries; i++ {
		req := types.BeaconBlockByRootsReq(roots)
		if len(roots) > int(params.BeaconNetworkConfig().MaxRequestBlocks) {
			req = roots[:params.BeaconNetworkConfig().MaxRequestBlocks]
		}
		if err := s.sendRecentBeaconBlocksRequest(ctx, &req, pid); err != nil {
			traceutil.AnnotateError(span, err)
			log.Debugf("Could not send recent block request: %v", err)
		}
		newRoots := make([][32]byte, 0, len(roots))
		s.pendingQueueLock.RLock()
		for _, rt := range roots {
			if !s.seenPendingBlocks[rt] {
				newRoots = append(newRoots, rt)
			}
		}
		s.pendingQueueLock.RUnlock()
		if len(newRoots) == 0 {
			break
		}
		// Choosing a new peer with the leftover set of
		// roots to request.
		roots = newRoots
		pid = bestPeers[randGen.Int()%len(bestPeers)]
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
	for slot, blks := range s.slotToPendingBlocks {
		for _, b := range blks {
			epoch := helpers.SlotToEpoch(slot)
			// remove all descendant blocks of old blocks
			if oldBlockRoots[bytesutil.ToBytes32(b.Block.ParentRoot)] {
				root, err := b.Block.HashTreeRoot()
				if err != nil {
					return err
				}
				oldBlockRoots[root] = true
				s.deleteBlockFromPendingQueue(slot, b, root)
				continue
			}
			// don't process old blocks
			if finalizedEpoch > 0 && epoch <= finalizedEpoch {
				blkRoot, err := b.Block.HashTreeRoot()
				if err != nil {
					return err
				}
				oldBlockRoots[blkRoot] = true
				s.deleteBlockFromPendingQueue(slot, b, blkRoot)
			}
		}
	}
	return nil
}

func (s *Service) clearPendingSlots() {
	s.pendingQueueLock.Lock()
	defer s.pendingQueueLock.Unlock()
	s.slotToPendingBlocks = make(map[uint64][]*ethpb.SignedBeaconBlock)
	s.seenPendingBlocks = make(map[[32]byte]bool)
}

// Delete block from the list from the pending queue using the slot as key.
// Note: this helper is not thread safe.
func (s *Service) deleteBlockFromPendingQueue(slot uint64, b *ethpb.SignedBeaconBlock, r [32]byte) {
	blks, ok := s.slotToPendingBlocks[slot]
	if !ok {
		return
	}
	newBlks := make([]*ethpb.SignedBeaconBlock, 0, len(blks))
	for _, blk := range blks {
		if ssz.DeepEqual(blk, b) {
			continue
		}
		newBlks = append(newBlks, blk)
	}
	if len(newBlks) == 0 {
		delete(s.slotToPendingBlocks, slot)
		return
	}
	s.slotToPendingBlocks[slot] = newBlks
	delete(s.seenPendingBlocks, r)
}

// Insert block to the list in the pending queue using the slot as key.
// Note: this helper is not thread safe.
func (s *Service) insertBlockToPendingQueue(slot uint64, b *ethpb.SignedBeaconBlock, r [32]byte) {
	if s.seenPendingBlocks[r] {
		return
	}

	_, ok := s.slotToPendingBlocks[slot]
	if ok {
		blks := s.slotToPendingBlocks[slot]
		s.slotToPendingBlocks[slot] = append(blks, b)
	} else {
		s.slotToPendingBlocks[slot] = []*ethpb.SignedBeaconBlock{b}
	}
	s.seenPendingBlocks[r] = true
}
