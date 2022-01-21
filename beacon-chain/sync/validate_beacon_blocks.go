package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (s *Service) validateBeaconBlockPubSub(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	receivedTime := prysmTime.Now()
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}

	// We should not attempt to process blocks until fully synced, but propagation is OK.
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, errors.Wrap(err, "Could not decode message")
	}

	s.validateBlockLock.Lock()
	defer s.validateBlockLock.Unlock()

	blk, ok := m.(block.SignedBeaconBlock)
	if !ok {
		return pubsub.ValidationReject, errors.New("msg is not ethpb.SignedBeaconBlock")
	}

	if blk.IsNil() || blk.Block().IsNil() {
		return pubsub.ValidationReject, errors.New("block.Block is nil")
	}

	// Broadcast the block on a feed to notify other services in the beacon node
	// of a received block (even if it does not process correctly through a state transition).
	s.cfg.blockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.ReceivedBlock,
		Data: &blockfeed.ReceivedBlockData{
			SignedBlock: blk,
		},
	})

	if features.Get().EnableSlasher {
		// Feed the block header to slasher if enabled. This action
		// is done in the background to avoid adding more load to this critical code path.
		go func() {
			blockHeader, err := block.SignedBeaconBlockHeaderFromBlockInterface(blk)
			if err != nil {
				log.WithError(err).WithField("blockSlot", blk.Block().Slot()).Warn("Could not extract block header")
			}
			s.cfg.slasherBlockHeadersFeed.Send(blockHeader)
		}()
	}

	// Verify the block is the first block received for the proposer for the slot.
	if s.hasSeenBlockIndexSlot(blk.Block().Slot(), blk.Block().ProposerIndex()) {
		return pubsub.ValidationIgnore, nil
	}

	blockRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block().Slot()).Debug("Ignored block")
		return pubsub.ValidationIgnore, nil
	}
	if s.cfg.beaconDB.HasBlock(ctx, blockRoot) {
		return pubsub.ValidationIgnore, nil
	}
	// Check if parent is a bad block and then reject the block.
	if s.hasBadBlock(bytesutil.ToBytes32(blk.Block().ParentRoot())) {
		s.setBadBlock(ctx, blockRoot)
		e := fmt.Errorf("received block with root %#x that has an invalid parent %#x", blockRoot, blk.Block().ParentRoot())
		return pubsub.ValidationReject, e
	}

	s.pendingQueueLock.RLock()
	if s.seenPendingBlocks[blockRoot] {
		s.pendingQueueLock.RUnlock()
		return pubsub.ValidationIgnore, nil
	}
	s.pendingQueueLock.RUnlock()

	// Be lenient in handling early blocks. Instead of discarding blocks arriving later than
	// MAXIMUM_GOSSIP_CLOCK_DISPARITY in future, we tolerate blocks arriving at max two slots
	// earlier (SECONDS_PER_SLOT * 2 seconds). Queue such blocks and process them at the right slot.
	genesisTime := uint64(s.cfg.chain.GenesisTime().Unix())
	if err := slots.VerifyTime(genesisTime, blk.Block().Slot(), earlyBlockProcessingTolerance); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block().Slot()).Debug("Ignored block")
		return pubsub.ValidationIgnore, nil
	}

	// Add metrics for block arrival time subtracts slot start time.
	if err := captureArrivalTimeMetric(genesisTime, blk.Block().Slot()); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block().Slot()).Debug("Ignored block")
		return pubsub.ValidationIgnore, nil
	}

	startSlot, err := slots.EpochStart(s.cfg.chain.FinalizedCheckpt().Epoch)
	if err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block().Slot()).Debug("Ignored block")
		return pubsub.ValidationIgnore, nil
	}
	if startSlot >= blk.Block().Slot() {
		e := fmt.Errorf("finalized slot %d greater or equal to block slot %d", startSlot, blk.Block().Slot())
		return pubsub.ValidationIgnore, e
	}

	// Process the block if the clock jitter is less than MAXIMUM_GOSSIP_CLOCK_DISPARITY.
	// Otherwise queue it for processing in the right slot.
	if isBlockQueueable(genesisTime, blk.Block().Slot(), receivedTime) {
		s.pendingQueueLock.Lock()
		if err := s.insertBlockToPendingQueue(blk.Block().Slot(), blk, blockRoot); err != nil {
			s.pendingQueueLock.Unlock()
			return pubsub.ValidationIgnore, err
		}
		s.pendingQueueLock.Unlock()
		e := fmt.Errorf("early block, with current slot %d < block slot %d", s.cfg.chain.CurrentSlot(), blk.Block().Slot())
		return pubsub.ValidationIgnore, e
	}

	// Handle block when the parent is unknown.
	if !s.cfg.beaconDB.HasBlock(ctx, bytesutil.ToBytes32(blk.Block().ParentRoot())) && !s.cfg.chain.HasInitSyncBlock(bytesutil.ToBytes32(blk.Block().ParentRoot())) {
		s.pendingQueueLock.Lock()
		if err := s.insertBlockToPendingQueue(blk.Block().Slot(), blk, blockRoot); err != nil {
			s.pendingQueueLock.Unlock()
			return pubsub.ValidationIgnore, err
		}
		s.pendingQueueLock.Unlock()
		return pubsub.ValidationIgnore, errors.Errorf("unknown parent for block with slot %d and parent root %#x", blk.Block().Slot(), blk.Block().ParentRoot())
	}

	if err := s.validateBeaconBlock(ctx, blk, blockRoot); err != nil {
		return pubsub.ValidationReject, err
	}

	// Record attribute of valid block.
	span.AddAttributes(trace.Int64Attribute("slotInEpoch", int64(blk.Block().Slot()%params.BeaconConfig().SlotsPerEpoch)))
	msg.ValidatorData = blk.Proto() // Used in downstream subscriber

	// Log the arrival time of the accepted block
	startTime, err := slots.ToTime(genesisTime, blk.Block().Slot())
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	log.WithFields(logrus.Fields{
		"blockSlot":          blk.Block().Slot(),
		"sinceSlotStartTime": receivedTime.Sub(startTime),
	}).Debug("Received block")
	return pubsub.ValidationAccept, nil
}

func (s *Service) validateBeaconBlock(ctx context.Context, blk block.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlock")
	defer span.End()

	if err := s.cfg.chain.VerifyBlkDescendant(ctx, bytesutil.ToBytes32(blk.Block().ParentRoot())); err != nil {
		s.setBadBlock(ctx, blockRoot)
		return err
	}

	hasStateSummaryDB := s.cfg.beaconDB.HasStateSummary(ctx, bytesutil.ToBytes32(blk.Block().ParentRoot()))
	if !hasStateSummaryDB {
		_, err := s.cfg.stateGen.RecoverStateSummary(ctx, bytesutil.ToBytes32(blk.Block().ParentRoot()))
		if err != nil {
			return err
		}
	}
	parentState, err := s.cfg.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(blk.Block().ParentRoot()))
	if err != nil {
		return err
	}

	if err := blocks.VerifyBlockSignatureUsingCurrentFork(parentState, blk); err != nil {
		s.setBadBlock(ctx, blockRoot)
		return err
	}
	// In the event the block is more than an epoch ahead from its
	// parent state, we have to advance the state forward.
	parentState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, parentState, blk.Block().ParentRoot(), blk.Block().Slot())
	if err != nil {
		return err
	}
	idx, err := helpers.BeaconProposerIndex(ctx, parentState)
	if err != nil {
		return err
	}
	if blk.Block().ProposerIndex() != idx {
		s.setBadBlock(ctx, blockRoot)
		return errors.New("incorrect proposer index")
	}

	return nil
}

// Returns true if the block is not the first block proposed for the proposer for the slot.
func (s *Service) hasSeenBlockIndexSlot(slot types.Slot, proposerIdx types.ValidatorIndex) bool {
	s.seenBlockLock.RLock()
	defer s.seenBlockLock.RUnlock()
	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(proposerIdx))...)
	_, seen := s.seenBlockCache.Get(string(b))
	return seen
}

// Set block proposer index and slot as seen for incoming blocks.
func (s *Service) setSeenBlockIndexSlot(slot types.Slot, proposerIdx types.ValidatorIndex) {
	s.seenBlockLock.Lock()
	defer s.seenBlockLock.Unlock()
	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(proposerIdx))...)
	s.seenBlockCache.Add(string(b), true)
}

// Returns true if the block is marked as a bad block.
func (s *Service) hasBadBlock(root [32]byte) bool {
	s.badBlockLock.RLock()
	defer s.badBlockLock.RUnlock()
	_, seen := s.badBlockCache.Get(string(root[:]))
	return seen
}

// Set bad block in the cache.
func (s *Service) setBadBlock(ctx context.Context, root [32]byte) {
	s.badBlockLock.Lock()
	defer s.badBlockLock.Unlock()
	if ctx.Err() != nil { // Do not mark block as bad if it was due to context error.
		return
	}
	s.badBlockCache.Add(string(root[:]), true)
}

// This captures metrics for block arrival time by subtracts slot start time.
func captureArrivalTimeMetric(genesisTime uint64, currentSlot types.Slot) error {
	startTime, err := slots.ToTime(genesisTime, currentSlot)
	if err != nil {
		return err
	}
	ms := prysmTime.Now().Sub(startTime) / time.Millisecond
	arrivalBlockPropagationHistogram.Observe(float64(ms))

	return nil
}

// isBlockQueueable checks if the slot_time in the block is greater than
// current_time +  MAXIMUM_GOSSIP_CLOCK_DISPARITY. in short, this function
// returns true if the corresponding block should be queued and false if
// the block should be processed immediately.
func isBlockQueueable(genesisTime uint64, slot types.Slot, receivedTime time.Time) bool {
	slotTime, err := slots.ToTime(genesisTime, slot)
	if err != nil {
		return false
	}

	currentTimeWithDisparity := receivedTime.Add(params.BeaconNetworkConfig().MaximumGossipClockDisparity)
	return currentTimeWithDisparity.Unix() < slotTime.Unix()
}
