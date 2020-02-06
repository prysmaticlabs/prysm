package sync

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func (r *Service) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	signed := msg.(*ethpb.SignedBeaconBlock)

	if signed == nil || signed.Block == nil {
		return errors.New("nil block")
	}

	block := signed.Block

	headState, err := r.chain.HeadState(ctx)
	if err != nil {
		log.Errorf("Head state is not available: %v", err)
		return nil
	}
	// Ignore block older than last finalized checkpoint.
	if block.Slot < helpers.StartSlot(headState.FinalizedCheckpointEpoch()) {
		log.Debugf("Received a block older than finalized checkpoint, %d < %d",
			block.Slot, helpers.StartSlot(headState.FinalizedCheckpointEpoch()))
		return nil
	}

	blockRoot, err := ssz.HashTreeRoot(block)
	if err != nil {
		log.Errorf("Could not sign root block: %v", err)
		return nil
	}

	if r.db.HasBlock(ctx, blockRoot) {
		return nil
	}

	// Handle block when the parent is unknown
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(block.ParentRoot)) {
		r.pendingQueueLock.Lock()
		r.slotToPendingBlocks[block.Slot] = signed
		r.seenPendingBlocks[blockRoot] = true
		r.pendingQueueLock.Unlock()
		return nil
	}

	// Broadcast the block on a feed to notify other services in the beacon node
	// of a received block (even if it does not process correctly through a state transition).
	r.blockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.ReceivedBlock,
		Data: &blockfeed.ReceivedBlockData{
			SignedBlock: signed,
		},
	})

	err = r.chain.ReceiveBlockNoPubsub(ctx, signed)
	if err != nil {
		interop.WriteBlockToDisk(signed, true /*failed*/)
	}

	// Delete attestations from the block in the pool to avoid inclusion in future block.
	if err := r.deleteAttsInPool(block.Body.Attestations); err != nil {
		log.Errorf("Could not delete attestations in pool: %v", err)
		return nil
	}

	return err
}

// The input attestations are seen by the network, this deletes them from pool
// so proposers don't include them in a block for the future.
func (r *Service) deleteAttsInPool(atts []*ethpb.Attestation) error {
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
