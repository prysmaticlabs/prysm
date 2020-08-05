package sync

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (s *Service) validateBeaconBlockPubSub(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.p2p.PeerID() {
		return pubsub.ValidationAccept
	}

	// We should not attempt to process blocks until fully synced, but propagation is OK.
	if s.initialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	s.validateBlockLock.Lock()
	defer s.validateBlockLock.Unlock()

	blk, ok := m.(*ethpb.SignedBeaconBlock)
	if !ok {
		return pubsub.ValidationReject
	}

	if blk.Block == nil {
		return pubsub.ValidationReject
	}

	// Broadcast the block on a feed to notify other services in the beacon node
	// of a received block (even if it does not process correctly through a state transition).
	s.blockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.ReceivedBlock,
		Data: &blockfeed.ReceivedBlockData{
			SignedBlock: blk,
		},
	})

	// Verify the block is the first block received for the proposer for the slot.
	if s.hasSeenBlockIndexSlot(blk.Block.Slot, blk.Block.ProposerIndex) {
		return pubsub.ValidationIgnore
	}

	blockRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		return pubsub.ValidationIgnore
	}
	if s.db.HasBlock(ctx, blockRoot) {
		return pubsub.ValidationIgnore
	}
	// Check if parent is a bad block and then reject the block.
	if s.hasBadBlock(bytesutil.ToBytes32(blk.Block.ParentRoot)) {
		log.Debugf("Received block with root %#x that has an invalid parent %#x", blockRoot, blk.Block.ParentRoot)
		s.setBadBlock(blockRoot)
		return pubsub.ValidationReject
	}

	s.pendingQueueLock.RLock()
	if s.seenPendingBlocks[blockRoot] {
		s.pendingQueueLock.RUnlock()
		return pubsub.ValidationIgnore
	}
	s.pendingQueueLock.RUnlock()

	// Add metrics for block arrival time subtracts slot start time.
	if captureArrivalTimeMetric(uint64(s.chain.GenesisTime().Unix()), blk.Block.Slot) != nil {
		return pubsub.ValidationIgnore
	}

	if err := helpers.VerifySlotTime(uint64(s.chain.GenesisTime().Unix()), blk.Block.Slot, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Rejecting incoming block.")
		return pubsub.ValidationIgnore
	}

	if helpers.StartSlot(s.chain.FinalizedCheckpt().Epoch) >= blk.Block.Slot {
		log.Debug("Block slot older/equal than last finalized epoch start slot, rejecting it")
		return pubsub.ValidationIgnore
	}

	// Handle block when the parent is unknown.
	if !s.db.HasBlock(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot)) {
		s.pendingQueueLock.Lock()
		if len(s.slotToPendingBlocks) < 2000 {
			s.slotToPendingBlocks[blk.Block.Slot] = blk
			s.seenPendingBlocks[blockRoot] = true
		}
		s.pendingQueueLock.Unlock()
		return pubsub.ValidationIgnore
	}

	if err := s.chain.VerifyBlkDescendant(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot)); err != nil {
		log.WithError(err).Warn("Rejecting block")
		s.setBadBlock(blockRoot)
		return pubsub.ValidationReject
	}

	hasStateSummaryDB := s.db.HasStateSummary(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot))
	hasStateSummaryCache := s.stateSummaryCache.Has(bytesutil.ToBytes32(blk.Block.ParentRoot))
	if !hasStateSummaryDB && !hasStateSummaryCache {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("No access to parent state")
		return pubsub.ValidationIgnore
	}

	parentState, err := s.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot))
	if err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not get parent state")
		return pubsub.ValidationIgnore
	}

	if err := blocks.VerifyBlockSignature(parentState, blk); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not verify block signature")
		s.setBadBlock(blockRoot)
		return pubsub.ValidationReject
	}

	parentState, err = state.ProcessSlots(context.Background(), parentState, blk.Block.Slot)
	if err != nil {
		log.Errorf("Could not advance slot to calculate proposer index: %v", err)
		return pubsub.ValidationIgnore
	}
	idx, err := helpers.BeaconProposerIndex(parentState)
	if err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not get proposer index using parent state")
		return pubsub.ValidationIgnore
	}
	if blk.Block.ProposerIndex != idx {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Incorrect proposer index")
		s.setBadBlock(blockRoot)
		return pubsub.ValidationReject
	}

	msg.ValidatorData = blk // Used in downstream subscriber
	return pubsub.ValidationAccept
}

// Returns true if the block is not the first block proposed for the proposer for the slot.
func (s *Service) hasSeenBlockIndexSlot(slot uint64, proposerIdx uint64) bool {
	s.seenBlockLock.RLock()
	defer s.seenBlockLock.RUnlock()
	b := append(bytesutil.Bytes32(slot), bytesutil.Bytes32(proposerIdx)...)
	_, seen := s.seenBlockCache.Get(string(b))
	return seen
}

// Set block proposer index and slot as seen for incoming blocks.
func (s *Service) setSeenBlockIndexSlot(slot uint64, proposerIdx uint64) {
	s.seenBlockLock.Lock()
	defer s.seenBlockLock.Unlock()
	b := append(bytesutil.Bytes32(slot), bytesutil.Bytes32(proposerIdx)...)
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
func (s *Service) setBadBlock(root [32]byte) {
	s.badBlockLock.Lock()
	defer s.badBlockLock.Unlock()
	s.badBlockCache.Add(string(root[:]), true)
}

// This captures metrics for block arrival time by subtracts slot start time.
func captureArrivalTimeMetric(genesisTime uint64, currentSlot uint64) error {
	startTime, err := helpers.SlotToTime(genesisTime, currentSlot)
	if err != nil {
		return err
	}
	diffMs := roughtime.Now().Sub(startTime) / time.Millisecond
	arrivalBlockPropagationHistogram.Observe(float64(diffMs))

	return nil
}
