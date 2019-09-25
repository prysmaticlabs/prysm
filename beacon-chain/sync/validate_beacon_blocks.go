package sync

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// recentlySeenBlockRoots cache with max size of ~3Mib
var recentlySeenRoots = ccache.New(ccache.Configure().MaxSize(100000))

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (r *RegularSync) validateBeaconBlockPubSub(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) bool {
	r.validateBlockLock.Lock()
	defer r.validateBlockLock.Unlock()
	m := msg.(*ethpb.BeaconBlock)

	blockRoot, err := ssz.SigningRoot(m)
	if err != nil {
		log.WithField("validate", "beacon block").WithError(err).Error("Failed to get signing root of block")
		return false
	}

	if r.seenPendingBlocks[blockRoot] {
		return false
	}
	if recentlySeenRoots.Get(string(blockRoot[:])) != nil || r.db.HasBlock(ctx, blockRoot) {
		return false
	}
	recentlySeenRoots.Set(string(blockRoot[:]), true /*value*/, 365*24*time.Hour /*TTL*/)

	if fromSelf {
		return false
	}

	_, err = bls.SignatureFromBytes(m.Signature)
	if err == nil {
		p.Broadcast(ctx, m)
	}

	// We should not attempt to process blocks until fully synced, but propagation is OK.
	if r.initialSync.Syncing() {
		return false
	}

	return err == nil
}
