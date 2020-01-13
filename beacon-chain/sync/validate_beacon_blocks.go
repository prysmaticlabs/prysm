package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
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

	blockRoot, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		return false
	}

	r.pendingQueueLock.RLock()
	if r.seenPendingBlocks[blockRoot] {
		r.pendingQueueLock.RUnlock()
		return false
	}
	r.pendingQueueLock.RUnlock()

	if err := helpers.VerifySlotTime(uint64(r.chain.GenesisTime().Unix()), blk.Block.Slot); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Block.Slot).Warn("Rejecting incoming block.")
		return false
	}

	if r.chain.FinalizedCheckpt().Epoch > helpers.SlotToEpoch(blk.Block.Slot) {
		log.Debug("Block older than finalized checkpoint received,rejecting it")
		return false
	}

	if _, err = bls.SignatureFromBytes(blk.Signature); err != nil {
		return false
	}

	msg.ValidatorData = blk // Used in downstream subscriber
	return true
}
