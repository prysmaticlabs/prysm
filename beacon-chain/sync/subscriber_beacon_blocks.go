package sync

import (
	"context"
	"errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
)

func (r *Service) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	signed, ok := msg.(*ethpb.SignedBeaconBlock)
	if !ok {
		return errors.New("message is not type *ethpb.SignedBeaconBlock")
	}

	if signed == nil || signed.Block == nil {
		return errors.New("nil block")
	}

	r.setSeenBlockIndexSlot(signed.Block.Slot, signed.Block.ProposerIndex)

	block := signed.Block

	root, err := stateutil.BlockRoot(block)
	if err != nil {
		return err
	}

	// Broadcast the block on a feed to notify other services in the beacon node
	// of a received block (even if it does not process correctly through a state transition).
	r.blockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.ReceivedBlock,
		Data: &blockfeed.ReceivedBlockData{
			SignedBlock: signed,
		},
	})

	if err := r.chain.ReceiveBlockNoPubsub(ctx, signed, root); err != nil {
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
