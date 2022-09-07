package sync

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	p2ptypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/rand"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/equality"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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
		// Don't process the pending blocks if genesis time has not been set. The chain is not ready.
		if !s.isGenesisTimeSet() {
			return
		}
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
	ss := s.sortedPendingSlots()
	var parentRoots [][32]byte

	span.AddAttributes(
		trace.Int64Attribute("numSlots", int64(len(ss))),
		trace.Int64Attribute("numPeers", int64(len(pids))),
	)

	randGen := rand.NewGenerator()
	for _, slot := range ss {
		// process the blocks during their respective slot.
		// otherwise wait for the right slot to process the block.
		if slot > s.cfg.chain.CurrentSlot() {
			continue
		}

		ctx, span := trace.StartSpan(ctx, "processPendingBlocks.InnerLoop")
		span.AddAttributes(trace.Int64Attribute("slot", int64(slot))) // lint:ignore uintcast -- This conversion is OK for tracing.

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
				if err = s.deleteBlockFromPendingQueue(slot, b, blkRoot); err != nil {
					s.pendingQueueLock.Unlock()
					return err
				}
				s.pendingQueueLock.Unlock()
				span.End()
				continue
			}

			s.pendingQueueLock.RLock()
			inPendingQueue := s.seenPendingBlocks[b.Block().ParentRoot()]
			s.pendingQueueLock.RUnlock()

			keepProcessing, err := s.checkIfBlockIsBad(ctx, span, slot, b, blkRoot)
			if err != nil {
				return err
			}
			if !keepProcessing {
				continue
			}

			parentInDb := s.cfg.beaconDB.HasBlock(ctx, b.Block().ParentRoot())
			hasPeer := len(pids) != 0

			// Only request for missing parent block if it's not in beaconDB, not in pending cache
			// and has peer in the peer list.
			parentRoot := b.Block().ParentRoot()
			if !inPendingQueue && !parentInDb && hasPeer {
				log.WithFields(logrus.Fields{
					"currentSlot": b.Block().Slot(),
					"parentRoot":  hex.EncodeToString(bytesutil.Trunc(parentRoot[:])),
				}).Debug("Requesting parent block")
				parentRoots = append(parentRoots, b.Block().ParentRoot())

				span.End()
				continue
			}

			if !parentInDb {
				span.End()
				continue
			}

			err = s.validateBeaconBlock(ctx, b, blkRoot)
			switch {
			case errors.Is(ErrOptimisticParent, err): // Ok to continue process block with parent that is an optimistic candidate.
			case err != nil:
				log.WithError(err).WithField("slot", b.Block().Slot()).Debug("Could not validate block")
				s.setBadBlock(ctx, blkRoot)
				tracing.AnnotateError(span, err)
				span.End()
				continue
			default:
			}

			if err := s.cfg.chain.ReceiveBlock(ctx, b, blkRoot); err != nil {
				if blockchain.IsInvalidBlock(err) {
					r := blockchain.InvalidBlockRoot(err)
					if r != [32]byte{} {
						s.setBadBlock(ctx, r) // Setting head block as bad.
					} else {
						s.setBadBlock(ctx, blkRoot)
					}
				}
				log.WithError(err).WithField("slot", b.Block().Slot()).Debug("Could not process block")

				// In the next iteration of the queue, this block will be removed from
				// the pending queue as it has been marked as a 'bad' block.
				span.End()
				continue
			}

			s.setSeenBlockIndexSlot(b.Block().Slot(), b.Block().ProposerIndex())

			// Broadcasting the block again once a node is able to process it.
			pb, err := b.Proto()
			if err != nil {
				log.WithError(err).Debug("Could not get protobuf block")
			} else {
				if err := s.cfg.p2p.Broadcast(ctx, pb); err != nil {
					log.WithError(err).Debug("Could not broadcast block")
				}
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

func (s *Service) checkIfBlockIsBad(
	ctx context.Context,
	span *trace.Span,
	slot types.Slot,
	b interfaces.SignedBeaconBlock,
	blkRoot [32]byte,
) (keepProcessing bool, err error) {
	parentIsBad := s.hasBadBlock(b.Block().ParentRoot())
	blockIsBad := s.hasBadBlock(blkRoot)
	// Check if parent is a bad block.
	if parentIsBad || blockIsBad {
		// Set block as bad if its parent block is bad too.
		if parentIsBad {
			s.setBadBlock(ctx, blkRoot)
		}
		// Remove block from queue.
		s.pendingQueueLock.Lock()
		if err = s.deleteBlockFromPendingQueue(slot, b, blkRoot); err != nil {
			s.pendingQueueLock.Unlock()
			return false, err
		}
		s.pendingQueueLock.Unlock()
		span.End()
		return false, nil
	}

	return true, nil
}

func (s *Service) sendBatchRootRequest(ctx context.Context, roots [][32]byte, randGen *rand.Rand) error {
	ctx, span := trace.StartSpan(ctx, "sendBatchRootRequest")
	defer span.End()

	if len(roots) == 0 {
		return nil
	}
	cp := s.cfg.chain.FinalizedCheckpt()
	_, bestPeers := s.cfg.p2p.Peers().BestFinalized(maxPeerRequest, cp.Epoch)
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
			log.WithError(err).Debug("Could not send recent block request")
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

	ss := make([]types.Slot, 0, len(items))
	for k := range items {
		slot := cacheKeyToSlot(k)
		ss = append(ss, slot)
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i] < ss[j]
	})
	return ss
}

// validatePendingSlots validates the pending blocks
// by their slot. If they are before the current finalized
// checkpoint, these blocks are removed from the queue.
func (s *Service) validatePendingSlots() error {
	s.pendingQueueLock.Lock()
	defer s.pendingQueueLock.Unlock()
	oldBlockRoots := make(map[[32]byte]bool)

	cp := s.cfg.chain.FinalizedCheckpt()
	finalizedEpoch := cp.Epoch
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
			if oldBlockRoots[b.Block().ParentRoot()] {
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
func (s *Service) deleteBlockFromPendingQueue(slot types.Slot, b interfaces.SignedBeaconBlock, r [32]byte) error {
	mutexasserts.AssertRWMutexLocked(&s.pendingQueueLock)

	blks := s.pendingBlocksInCache(slot)
	if len(blks) == 0 {
		return nil
	}

	// Defensive check to ignore nil blocks
	if err := blocks.BeaconBlockIsNil(b); err != nil {
		return err
	}

	newBlks := make([]interfaces.SignedBeaconBlock, 0, len(blks))
	for _, blk := range blks {
		blkPb, err := blk.Proto()
		if err != nil {
			return err
		}
		bPb, err := b.Proto()
		if err != nil {
			return err
		}
		if equality.DeepEqual(blkPb, bPb) {
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
func (s *Service) insertBlockToPendingQueue(_ types.Slot, b interfaces.SignedBeaconBlock, r [32]byte) error {
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
func (s *Service) pendingBlocksInCache(slot types.Slot) []interfaces.SignedBeaconBlock {
	k := slotToCacheKey(slot)
	value, ok := s.slotToPendingBlocks.Get(k)
	if !ok {
		return []interfaces.SignedBeaconBlock{}
	}
	blks, ok := value.([]interfaces.SignedBeaconBlock)
	if !ok {
		return []interfaces.SignedBeaconBlock{}
	}
	return blks
}

// This adds input signed beacon block to slotToPendingBlocks cache.
func (s *Service) addPendingBlockToCache(b interfaces.SignedBeaconBlock) error {
	if err := blocks.BeaconBlockIsNil(b); err != nil {
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

// Returns true if the genesis time has been set in chain service.
// Without the genesis time, the chain does not start.
func (s *Service) isGenesisTimeSet() bool {
	return s.cfg.chain.GenesisTime().Unix() != 0
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
