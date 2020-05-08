package sync

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (r *Service) validateBeaconBlockPubSub(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == r.p2p.PeerID() {
		return true
	}

	// We should not attempt to process blocks until fully synced, but propagation is OK.
	if r.initialSync.Syncing() {
		return false
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	m, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return false
	}

	r.validateBlockLock.Lock()
	defer r.validateBlockLock.Unlock()

	blk, ok := m.(*ethpb.SignedBeaconBlock)
	if !ok {
		return false
	}

	if blk.Block == nil {
		return false
	}

	// Verify the block is the first block received for the proposer for the slot.
	if r.hasSeenBlockIndexSlot(blk.Block.Slot, blk.Block.ProposerIndex) {
		return false
	}

	blockRoot, err := stateutil.BlockRoot(blk.Block)
	if err != nil {
		return false
	}
	if r.db.HasBlock(ctx, blockRoot) {
		return false
	}

	r.pendingQueueLock.RLock()
	if r.seenPendingBlocks[blockRoot] {
		r.pendingQueueLock.RUnlock()
		return false
	}
	r.pendingQueueLock.RUnlock()

	// Add metrics for block arrival time subtracts slot start time.
	if captureArrivalTimeMetric(uint64(r.chain.GenesisTime().Unix()), blk.Block.Slot) != nil {
		return false
	}

	if err := helpers.VerifySlotTime(uint64(r.chain.GenesisTime().Unix()), blk.Block.Slot, maximumGossipClockDisparity); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Rejecting incoming block.")
		return false
	}

	if helpers.StartSlot(r.chain.FinalizedCheckpt().Epoch) >= blk.Block.Slot {
		log.Debug("Block slot older/equal than last finalized epoch start slot, rejecting it")
		return false
	}

	// Handle block when the parent is unknown.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot)) {
		r.pendingQueueLock.Lock()
		r.slotToPendingBlocks[blk.Block.Slot] = blk
		r.seenPendingBlocks[blockRoot] = true
		r.pendingQueueLock.Unlock()
		return false
	}

	if featureconfig.Get().NewStateMgmt {
		hasStateSummaryDB := r.db.HasStateSummary(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot))
		hasStateSummaryCache := r.stateSummaryCache.Has(bytesutil.ToBytes32(blk.Block.ParentRoot))
		if !hasStateSummaryDB && !hasStateSummaryCache {
			log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("No access to parent state")
			return false
		}
		parentState, err := r.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot))
		if err != nil {
			log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not get parent state")
			return false
		}

		if err := blocks.VerifyBlockSignature(parentState, blk); err != nil {
			log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not verify block signature")
			return false
		}

		err = parentState.SetSlot(blk.Block.Slot)
		if err != nil {
			log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not set parent state slot")
			return false
		}
		idx, err := helpers.BeaconProposerIndex(parentState)
		if err != nil {
			log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Could not get proposer index using parent state")
			return false
		}
		if blk.Block.ProposerIndex != idx {
			log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Incorrect proposer index")
			return false
		}
	}

	msg.ValidatorData = blk // Used in downstream subscriber
	return true
}

// Returns true if the block is not the first block proposed for the proposer for the slot.
func (r *Service) hasSeenBlockIndexSlot(slot uint64, proposerIdx uint64) bool {
	r.seenBlockLock.RLock()
	defer r.seenBlockLock.RUnlock()
	b := append(bytesutil.Bytes32(slot), bytesutil.Bytes32(proposerIdx)...)
	_, seen := r.seenBlockCache.Get(string(b))
	return seen
}

// Set block proposer index and slot as seen for incoming blocks.
func (r *Service) setSeenBlockIndexSlot(slot uint64, proposerIdx uint64) {
	r.seenBlockLock.Lock()
	defer r.seenBlockLock.Unlock()
	b := append(bytesutil.Bytes32(slot), bytesutil.Bytes32(proposerIdx)...)
	r.seenBlockCache.Add(string(b), true)
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
