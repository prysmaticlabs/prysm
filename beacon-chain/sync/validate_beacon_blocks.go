package sync

import (
	"context"
	"strings"

	"github.com/dgraph-io/ristretto"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

var recentlySeenRootsSize = int64(1 << 16)

// recentlySeenBlockRoots cache with max size of ~2Mib ( including keys)
var recentlySeenRoots, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: recentlySeenRootsSize,
	MaxCost:     recentlySeenRootsSize,
	BufferItems: 64,
})

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (r *Service) validateBeaconBlockPubSub(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	// We should not attempt to process blocks until fully synced, but propagation is OK.
	if r.initialSync.Syncing() {
		return false
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	topic := msg.TopicIDs[0]
	topic = strings.TrimSuffix(topic, r.p2p.Encoding().ProtocolSuffix())
	base, ok := p2p.GossipTopicMappings[topic]
	if !ok {
		return false
	}
	m := proto.Clone(base)
	if err := r.p2p.Encoding().Decode(msg.Data, m); err != nil {
		traceutil.AnnotateError(span, err)
		log.WithError(err).Warn("Failed to decode pubsub message")
		return false
	}

	r.validateBlockLock.Lock()
	defer r.validateBlockLock.Unlock()

	blk, ok := m.(*ethpb.BeaconBlock)
	if !ok {
		return false
	}

	blockRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		return false
	}

	r.pendingQueueLock.RLock()
	if r.seenPendingBlocks[blockRoot] {
		r.pendingQueueLock.RUnlock()
		return false
	}
	r.pendingQueueLock.RUnlock()

	// Reject messages from self.
	if pid == r.p2p.PeerID() {
		return false
	}

	if err := helpers.VerifySlotTime(uint64(r.chain.GenesisTime().Unix()), blk.Slot); err != nil {
		log.WithError(err).WithField("blockSlot", blk.Slot).Warn("Rejecting incoming block.")
		return false
	}

	if r.chain.FinalizedCheckpt().Epoch > helpers.SlotToEpoch(blk.Slot) {
		log.Debug("Block older than finalized checkpoint received,rejecting it")
		return false
	}

	if _, err = bls.SignatureFromBytes(blk.Signature); err != nil {
		return false
	}

	msg.VaidatorData = blk // Used in downstream subscriber
	return true
}
