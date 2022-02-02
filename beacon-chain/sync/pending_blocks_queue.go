package sync

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/async"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"github.com/trailofbits/go-mutexasserts"
	"go.opencensus.io/trace"
)

var processPendingBlocksPeriod = slots.DivideSlotBy(3 /* times per slot */)

const maxPeerRequest = 50
const numOfTries = 5
const maxBlocksPerSlot = 3

// processes pending blocks queue on every processPendingBlocksPeriod
func (s *Service) processPendingBlocksQueue() {
	// Prevents multiple queue processing goroutines (invoked by RunEvery) from contending for data.
	locker := new(sync.Mutex)
	async.RunEvery(s.ctx, processPendingBlocksPeriod, func() {
		locker.Lock()
		if err := s.processPendingBlocks(s.ctx); err != nil {
			log.WithError(err).Debug("Could not process pending blocks")
		}
		locker.Unlock()
	})
}

// processes the block tree inside the queue
func (s *Service) processPendingBlocks(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "processPendingBlocks")
	defer span.End()

	pids := s.cfg.p2p.Peers().Connected()
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
		// process the blocks during their respective slot.
		// otherwise wait for the right slot to process the block.
		if slot > s.cfg.chain.CurrentSlot() {
			continue
		}

		ctx, span := trace.StartSpan(ctx, "processPendingBlocks.InnerLoop")
		span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))

		s.pendingQueueLock.RLock()
		bs := s.pendingBlocksInCache(slot)
		// Skip if there's no block in the queue.
		if len(bs) == 0 {
			s.pendingQueueLock.RUnlock()
			span.End()
			continue
		}
		s.pendingQueueLock.RUnlock()

		// Loop through the pending queue and mark the potential parent blocks as seen.
		for _, b := range bs {
			if b == nil || b.IsNil() || b.Block().IsNil() {
				span.End()
				continue
			}

			blkRoot, err := b.Block().HashTreeRoot()
			if err != nil {
				tracing.AnnotateError(span, err)
				span.End()
				return err
			}
			inDB := s.cfg.beaconDB.HasBlock(ctx, blkRoot)
			// No need to process the same block twice.
			if inDB {
				s.pendingQueueLock.Lock()
				if err := s.deleteBlockFromPendingQueue(slot, b, blkRoot); err != nil {
					s.pendingQueueLock.Unlock()
					return err
				}
				s.pendingQueueLock.Unlock()
				span.End()
				continue
			}

			s.pendingQueueLock.RLock()
			inPendingQueue := s.seenPendingBlocks[bytesutil.ToBytes32(b.Block().ParentRoot())]
			s.pendingQueueLock.RUnlock()

			parentIsBad := s.hasBadBlock(bytesutil.ToBytes32(b.Block().ParentRoot()))
			blockIsBad := s.hasBadBlock(blkRoot)
			// Check if parent is a bad block.
			if parentIsBad || blockIsBad {
				// Set block as bad if its parent block is bad too.
				if parentIsBad {
					s.setBadBlock(ctx, blkRoot)
				}
				// Remove block from queue.
				s.pendingQueueLock.Lock()
				if err := s.deleteBlockFromPendingQueue(slot, b, blkRoot); err != nil {
					s.pendingQueueLock.Unlock()
					return err
				}
				s.pendingQueueLock.Unlock()
				span.End()
				continue
			}

			parentInDb := s.cfg.beaconDB.HasBlock(ctx, bytesutil.ToBytes32(b.Block().ParentRoot()))
			hasPeer := len(pids) != 0

			// Only request for missing parent block if it's not in beaconDB, not in pending cache
			// and has peer in the peer list.
			if !inPendingQueue && !parentInDb && hasPeer {
				log.WithFields(logrus.Fields{
					"currentSlot": b.Block().Slot(),
					"parentRoot":  hex.EncodeToString(bytesutil.Trunc(b.Block().ParentRoot())),
				}).Debug("Requesting parent block")
				parentRoots = append(parentRoots, bytesutil.ToBytes32(b.Block().ParentRoot()))

				span.End()
				continue
			}

			if !parentInDb {
				span.End()
				continue
			}

			if err := s.validateBeaconBlock(ctx, b, blkRoot); err != nil {
				log.Debugf("Could not validate block from slot %d: %v", b.Block().Slot(), err)
				s.setBadBlock(ctx, blkRoot)
				tracing.AnnotateError(span, err)
				// In the next iteration of the queue, this block will be removed from
				// the pending queue as it has been marked as a 'bad' block.
				span.End()
				continue
			}

			if err := s.cfg.chain.ReceiveBlock(ctx, b, blkRoot); err != nil {
				log.Debugf("Could not process block from slot %d: %v", b.Block().Slot(), err)
				s.setBadBlock(ctx, blkRoot)
				tracing.AnnotateError(span, err)
				// In the next iteration of the queue, this block will be removed from
				// the pending queue as it has been marked as a 'bad' block.
				span.End()
				continue
			}

			s.setSeenBlockIndexSlot(b.Block().Slot(), b.Block().ProposerIndex())

			// Broadcasting the block again once a node is able to process it.
			if err := s.cfg.p2p.Broadcast(ctx, b.Proto()); err != nil {
				log.WithError(err).Debug("Could not broadcast block")
			}

			s.pendingQueueLock.Lock()
			if err := s.deleteBlockFromPendingQueue(slot, b, blkRoot); err != nil {
				s.pendingQueueLock.Unlock()
				return err
			}
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

	_, bestPeers := s.cfg.p2p.Peers().BestFinalized(maxPeerRequest, s.cfg.chain.FinalizedCheckpt().Epoch)
	if len(bestPeers) == 0 {
		return nil
	}
	roots = s.dedupRoots(roots)
	// Randomly choose a peer to query from our best peers. If that peer cannot return
	// all the requested blocks, we randomly select another peer.
	pid := bestPeers[randGen.Int()%len(bestPeers)]
	for i := 0; i < numOfTries; i++ {
		req := p2ptypes.BeaconBlockByRootsReq(roots)
		if len(roots) > int(params.BeaconNetworkConfig().MaxRequestBlocks) {
			req = roots[:params.BeaconNetworkConfig().MaxRequestBlocks]
		}
		if err := s.sendRecentBeaconBlocksRequest(ctx, &req, pid); err != nil {
			tracing.AnnotateError(span, err)
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

func (s *Service) sortedPendingSlots() []types.Slot {
	s.pendingQueueLock.RLock()
	defer s.pendingQueueLock.RUnlock()

	items := s.slotToPendingBlocks.Items()

	slots := make([]types.Slot, 0, len(items))
	for k := range items {
		slot := cacheKeyToSlot(k)
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

	finalizedEpoch := s.cfg.chain.FinalizedCheckpt().Epoch
	if s.slotToPendingBlocks == nil {
		return errors.New("slotToPendingBlocks cache can't be nil")
	}
	items := s.slotToPendingBlocks.Items()
	for k := range items {
		slot := cacheKeyToSlot(k)
		blks := s.pendingBlocksInCache(slot)
		for _, b := range blks {
			epoch := slots.ToEpoch(slot)
			// remove all descendant blocks of old blocks
			if oldBlockRoots[bytesutil.ToBytes32(b.Block().ParentRoot())] {
				root, err := b.Block().HashTreeRoot()
				if err != nil {
					return err
				}
				oldBlockRoots[root] = true
				if err := s.deleteBlockFromPendingQueue(slot, b, root); err != nil {
					return err
				}
				continue
			}
			// don't process old blocks
			if finalizedEpoch > 0 && epoch <= finalizedEpoch {
				blkRoot, err := b.Block().HashTreeRoot()
				if err != nil {
					return err
				}
				oldBlockRoots[blkRoot] = true
				if err := s.deleteBlockFromPendingQueue(slot, b, blkRoot); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Service) clearPendingSlots() {
	s.pendingQueueLock.Lock()
	defer s.pendingQueueLock.Unlock()
	s.slotToPendingBlocks.Flush()
	s.seenPendingBlocks = make(map[[32]byte]bool)
}

// Delete block from the list from the pending queue using the slot as key.
// Note: this helper is not thread safe.
func (s *Service) deleteBlockFromPendingQueue(slot types.Slot, b block.SignedBeaconBlock, r [32]byte) error {
	mutexasserts.AssertRWMutexLocked(&s.pendingQueueLock)

	blks := s.pendingBlocksInCache(slot)
	if len(blks) == 0 {
		return nil
	}

	// Defensive check to ignore nil blocks
	if err := helpers.BeaconBlockIsNil(b); err != nil {
		return err
	}

	newBlks := make([]block.SignedBeaconBlock, 0, len(blks))
	for _, blk := range blks {
		if ssz.DeepEqual(blk.Proto(), b.Proto()) {
			continue
		}
		newBlks = append(newBlks, blk)
	}
	if len(newBlks) == 0 {
		s.slotToPendingBlocks.Delete(slotToCacheKey(slot))
		delete(s.seenPendingBlocks, r)
		return nil
	}

	// Decrease exp time in proportion to how many blocks are still in the cache for slot key.
	d := pendingBlockExpTime / time.Duration(len(newBlks))
	if err := s.slotToPendingBlocks.Replace(slotToCacheKey(slot), newBlks, d); err != nil {
		return err
	}
	delete(s.seenPendingBlocks, r)
	return nil
}

// Insert block to the list in the pending queue using the slot as key.
// Note: this helper is not thread safe.
func (s *Service) insertBlockToPendingQueue(_ types.Slot, b block.SignedBeaconBlock, r [32]byte) error {
	mutexasserts.AssertRWMutexLocked(&s.pendingQueueLock)

	if s.seenPendingBlocks[r] {
		return nil
	}

	if err := s.addPendingBlockToCache(b); err != nil {
		return err
	}

	s.seenPendingBlocks[r] = true
	return nil
}

// This returns signed beacon blocks given input key from slotToPendingBlocks.
func (s *Service) pendingBlocksInCache(slot types.Slot) []block.SignedBeaconBlock {
	k := slotToCacheKey(slot)
	value, ok := s.slotToPendingBlocks.Get(k)
	if !ok {
		return []block.SignedBeaconBlock{}
	}
	blks, ok := value.([]block.SignedBeaconBlock)
	if !ok {
		return []block.SignedBeaconBlock{}
	}
	return blks
}

// This adds input signed beacon block to slotToPendingBlocks cache.
func (s *Service) addPendingBlockToCache(b block.SignedBeaconBlock) error {
	if err := helpers.BeaconBlockIsNil(b); err != nil {
		return err
	}

	blks := s.pendingBlocksInCache(b.Block().Slot())

	if len(blks) >= maxBlocksPerSlot {
		return nil
	}

	blks = append(blks, b)
	k := slotToCacheKey(b.Block().Slot())
	s.slotToPendingBlocks.Set(k, blks, pendingBlockExpTime)
	return nil
}

// This converts input string to slot.
func cacheKeyToSlot(s string) types.Slot {
	b := []byte(s)
	return bytesutil.BytesToSlotBigEndian(b)
}

// This converts input slot to a key to be used for slotToPendingBlocks cache.
func slotToCacheKey(s types.Slot) string {
	b := bytesutil.SlotToBytesBigEndian(s)
	return string(b)
}
