package beacon

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var lastSendSlot types.Slot

// StreamNewPendingBlocks to orchestrator client every single time an unconfirmed block is received by the beacon node.
func (bs *Server) StreamNewPendingBlocks(request *ethpb.StreamPendingBlocksRequest, stream ethpb.BeaconChain_StreamNewPendingBlocksServer) error {
	requestedSlot := request.FromSlot
	lastSendSlot = request.FromSlot
	cp, err := bs.BeaconDB.FinalizedCheckpoint(bs.Ctx)
	if err != nil {
		return status.Errorf(codes.Internal,
			"Could not retrieve finalize epoch: %v", err)
	}

	latestFinalizedEpoch := cp.Epoch
	latestFinalizedEndSlot, err := helpers.EndSlot(latestFinalizedEpoch)
	if err != nil {
		return status.Errorf(codes.Internal,
			"Could not retrieve start slot from latestFinalizedEpoch: %v", err)
	}

	if requestedSlot < latestFinalizedEndSlot {
		if err := bs.sendBlocksToLatestFinalizedEpoch(requestedSlot, latestFinalizedEpoch, stream); err != nil {
			return status.Errorf(codes.Internal,
				"Could not send previous blocks from requested slot to latest finalized epoch: %v", err)
		}
	}

	unconfirmedBlocksCh := make(chan *feed.Event, 1)
	unconfirmedBlocksSub := bs.BlockNotifier.BlockFeed().Subscribe(unconfirmedBlocksCh)
	defer unconfirmedBlocksSub.Unsubscribe()

	// Getting un-confirmed blocks from cache and sends those blocks to orchestrator
	unconfirmedBlocks, err := bs.UnconfirmedBlockFetcher.SortedUnConfirmedBlocksFromCache()
	if err != nil {
		return status.Errorf(codes.Unavailable,
			"Could not send cached un-confirmed blocks over stream: %v", err)
	}

	if len(unconfirmedBlocks) > 0 {
		toSlot := unconfirmedBlocks[0].Slot.Sub(1)
		if lastSendSlot < toSlot {
			if err := bs.sendBlocksFromFinalizedEpochToCurrent(lastSendSlot, toSlot, stream); err != nil {
				return status.Errorf(codes.Internal,
					"Could not send previous blocks from latest finalized epoch to current slot: %v", err)
			}
		}

		for _, blk := range unconfirmedBlocks {
			if err := stream.Send(blk); err != nil {
				return status.Errorf(codes.Unavailable,
					"Could not send un-confirmed block over stream: %v", err)
			}
		}
	}

	var flag bool
	for {
		select {
		case blockEvent := <-unconfirmedBlocksCh:
			if blockEvent.Type == blockfeed.UnConfirmedBlock {
				data, ok := blockEvent.Data.(*blockfeed.UnConfirmedBlockData)
				if !ok || data == nil {
					continue
				}

				if !flag {
					toSlot := data.Block.Slot.Sub(1)
					if lastSendSlot < toSlot {
						if err := bs.sendBlocksFromFinalizedEpochToCurrent(lastSendSlot.Add(1), toSlot, stream); err != nil {
							return status.Errorf(codes.Internal,
								"Could not send previous blocks from latest finalized epoch to current slot: %v", err)
						}
						flag = true
					}
				}

				if err := stream.Send(data.Block); err != nil {
					return status.Errorf(codes.Unavailable, "Could not send un-confirmed block over stream: %v", err)
				}

				log.WithField("slot", data.Block.Slot).Debug(
					"New pending block has been published successfully")
			}
		case <-unconfirmedBlocksSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}

// sendBlocksToLatestedFinalizedEpoch
func (bs *Server) sendBlocksToLatestFinalizedEpoch(
	requestedSlot types.Slot,
	finalizedEpoch types.Epoch,
	stream ethpb.BeaconChain_StreamNewPendingBlocksServer,
) error {

	startEpoch := helpers.SlotToEpoch(requestedSlot)
	for epoch := startEpoch; epoch <= finalizedEpoch; epoch++ {
		retrievedBlks, _, err := bs.BeaconDB.Blocks(bs.Ctx, filters.NewFilter().SetStartEpoch(epoch).SetEndEpoch(epoch))
		if err != nil {
			log.WithError(err).Warn("Failed to retrieve blocks from db")
			return err
		}
		for _, blk := range retrievedBlks {
			// we do not send block #0 to orchestrator
			if blk.Block.Slot == 0 {
				continue
			}

			if err := stream.Send(blk.Block); err != nil {
				return status.Errorf(codes.Unavailable, "Could not send previous blocks over stream: %v", err)
			}
		}
		endSlot, err := helpers.EndSlot(epoch)
		if err != nil {
			return err
		}
		lastSendSlot = endSlot
	}

	log.WithField("requestedEpoch", startEpoch).
		WithField("finalizedEpoch", finalizedEpoch).
		WithField("lastSendSlot", lastSendSlot).
		Debug("Successfully send previous finalized blocks from requested epoch to finalized epoch")

	return nil
}

// sendBlocksFromFinalizedEpochToCurrent
func (bs *Server) sendBlocksFromFinalizedEpochToCurrent(
	fromSlot types.Slot,
	toSlot types.Slot,
	stream ethpb.BeaconChain_StreamNewPendingBlocksServer) error {

	retrievedBlks, _, err := bs.BeaconDB.Blocks(bs.Ctx, filters.NewFilter().SetStartSlot(fromSlot).SetEndSlot(toSlot))
	if err != nil {
		log.WithError(err).Warn("Failed to retrieve blocks from latest finalized epoch to current slot")
		return err
	}
	for _, blk := range retrievedBlks {
		if err := stream.Send(blk.Block); err != nil {
			return status.Errorf(codes.Unavailable, "Could not send previous blocks from latest finalized epoch to current slot: %v", err)
		}
	}
	lastSendSlot = toSlot

	log.WithField("fromSlot", fromSlot).
		WithField("toSlot", toSlot).
		WithField("lastSendSlot", lastSendSlot).
		Debug("Successfully send previous blocks from finalized epoch to latest slot")

	return nil
}
