package sync

import (
	"context"
	"fmt"

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
func (r *RegularSync) validateBeaconBlockPubSub(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	r.validateBlockLock.Lock()
	defer r.validateBlockLock.Unlock()

	m, ok := msg.(*ethpb.BeaconBlock)
	if !ok {
		return false, fmt.Errorf("message was not type *eth.BeaconBlock, type=%T", msg)
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

	// Delete the same attestations from the block in the pool to avoid inclusion in future block.
	if err := r.deleteAttsInPool(m.Body.Attestations); err != nil {
		return false, err
	}

	if _, ok := recentlySeenRoots.Get(string(blockRoot[:])); ok || r.db.HasBlock(ctx, blockRoot) {
		return false, nil
	}
	recentlySeenRoots.Set(string(blockRoot[:]), true /*value*/, 1 /*cost*/)

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

// The input attestations are seen by the network, this deletes them from pool
// so proposers don't include them in a block for the future.
func (r *RegularSync) deleteAttsInPool(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := r.attPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			// Ideally there's shouldn't be any unaggregated attestation in the block.
			if err := r.attPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}
