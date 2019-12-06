package sync

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"go.opencensus.io/trace"
)

// recentlySeenBlockRoots cache with max size of ~2Mib ( including keys)
var recentlySeenRoots *ristretto.Cache
var recentlySeenRootsSize = int64(1 << 16)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (r *RegularSync) validateBeaconBlockPubSub(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	r.validateBlockLock.Lock()
	defer r.validateBlockLock.Unlock()

	m, ok := msg.(*ethpb.BeaconBlock)
	if !ok {
		return false, nil
	}

	blockRoot, err := ssz.SigningRoot(m)
	if err != nil {
		return false, errors.Wrap(err, "could not get signing root of beacon block")
	}

	r.pendingQueueLock.RLock()
	if r.seenPendingBlocks[blockRoot] {
		r.pendingQueueLock.RUnlock()
		return false, nil
	}
	r.pendingQueueLock.RUnlock()

	if recentlySeenRoots.Get(string(blockRoot[:])) != nil || r.db.HasBlock(ctx, blockRoot) {
		return false, nil
	}
	recentlySeenRoots.Set(string(blockRoot[:]), true /*value*/, 365*24*time.Hour /*TTL*/)

	if fromSelf {
		return false, nil
	}

	if err := helpers.VerifySlotTime(uint64(r.chain.GenesisTime().Unix()), m.Slot); err != nil {
		log.WithError(err).WithField("blockSlot", m.Slot).Warn("Rejecting incoming block.")
		return false, err
	}

	if r.chain.FinalizedCheckpt().Epoch > helpers.SlotToEpoch(m.Slot) {
		log.Debug("Block older than finalized checkpoint received,rejecting it")
		return false, nil
	}

	_, err = bls.SignatureFromBytes(m.Signature)
	if err == nil {
		p.Broadcast(ctx, m)
	}

	// We should not attempt to process blocks until fully synced, but propagation is OK.
	if r.initialSync.Syncing() {
		return false, nil
	}

	return err == nil, err
}
