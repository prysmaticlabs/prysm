package beacon

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamNewPendingBlocks to orchestrator client every single time an unconfirmed block is received by the beacon node.
func (bs *Server) StreamNewPendingBlocks(request *ethpb.StreamPendingBlocksRequest, stream ethpb.BeaconChain_StreamNewPendingBlocksServer) error {
	batchSender := func(start, end types.Epoch) error {
		for i := start; i <= end; i++ {
			blks, _, err := bs.BeaconDB.Blocks(bs.Ctx, filters.NewFilter().SetStartEpoch(i).SetEndEpoch(i))
			if err != nil {
				return status.Errorf(codes.Internal,
					"Could not send over stream: %v", err)
			}
			for _, blk := range blks {
				// we do not send block #0 to orchestrator
				if blk.Block().Slot() == 0 {
					continue
				}
				unwrappedBlk, err := blk.PbPhase0Block()
				if err != nil {
					return status.Errorf(codes.Internal,
						"Could not send over stream: %v", err)
				}
				if err := stream.Send(unwrappedBlk.Block); err != nil {
					return status.Errorf(codes.Unavailable,
						"Could not send over stream: %v", err)
				}
			}
		}
		return nil
	}

	sender := func(start, end types.Slot) error {
		blks, _, err := bs.BeaconDB.Blocks(bs.Ctx, filters.NewFilter().SetStartSlot(start).SetEndSlot(end))
		if err != nil {
			return err
		}
		for _, blk := range blks {
			unwrappedBlk, err := blk.PbPhase0Block()
			if err != nil {
				return status.Errorf(codes.Internal,
					"Could not send over stream: %v", err)
			}
			if err := stream.Send(unwrappedBlk.Block); err != nil {
				return status.Errorf(codes.Unavailable,
					"Could not send over stream: %v", err)
			}
		}
		return nil
	}

	cp, err := bs.BeaconDB.FinalizedCheckpoint(bs.Ctx)
	if err != nil {
		return status.Errorf(codes.Internal,
			"Could not retrieve finalize epoch: %v", err)
	}

	epochStart := helpers.SlotToEpoch(request.FromSlot)
	epochEnd := cp.Epoch

	if epochStart <= epochEnd {
		if err := batchSender(epochStart, epochEnd); err != nil {
			return err
		}
		log.WithField("startEpoch", epochStart).
			WithField("endEpoch", epochEnd).
			Info("Published previous blocks in batch")
	}
	// Getting un-confirmed blocks from cache and sends those blocks to orchestrator
	pBlocks, err := bs.UnconfirmedBlockFetcher.SortedUnConfirmedBlocksFromCache()
	if err != nil {
		return status.Errorf(codes.Internal,
			"Could not send over stream: %v", err)
	}

	startSlot, err := helpers.EndSlot(epochEnd)
	if err != nil {
		return status.Errorf(codes.Internal,
			"Could not retrieve end slot number: %v", err)
	}

	endSlot := startSlot

	if len(pBlocks) > 0 {
		for _, blk := range pBlocks {
			unwrappedBlk, err := blk.PbPhase0UnsignedBlock()
			if err != nil {
				return status.Errorf(codes.Internal,
					"Could not send over stream: %v", err)
			}
			if err := stream.Send(unwrappedBlk); err != nil {
				return status.Errorf(codes.Unavailable,
					"Could not send over stream: %v", err)
			}
			log.WithField("startSlot", pBlocks[0].Slot()).
				WithField("endSlot", pBlocks[len(pBlocks)-1].Slot()).
				Info("Published unconfirmed blocks")
		}
		// sending till unconfirmed first block's slot
		endSlot = pBlocks[0].Slot()
		if startSlot+1 < endSlot {
			if err := sender(startSlot, endSlot); err != nil {
				return err
			}
			log.WithField("startSlot", startSlot+1).
				WithField("endSlot", endSlot).
				Info("Published blocks from last sent epoch to unconfirmed block")
		}
	}

	pBlockCh := make(chan *feed.Event, 1)
	pBlockSub := bs.BlockNotifier.BlockFeed().Subscribe(pBlockCh)
	firstTime := true
	defer pBlockSub.Unsubscribe()

	for {
		select {
		case blockEvent := <-pBlockCh:
			if blockEvent.Type == blockfeed.UnConfirmedBlock {
				data, ok := blockEvent.Data.(*blockfeed.UnConfirmedBlockData)
				if !ok || data == nil {
					continue
				}

				if firstTime {
					firstTime = false
					startSlot = endSlot + 1
					endSlot = data.Block.Slot()
					log.WithField("startSlot", startSlot).
						WithField("endSlot", endSlot).
						WithField("liveSyncStart", endSlot+1).
						Info("Sending left over blocks")
					if startSlot < endSlot {
						if err := sender(startSlot, endSlot); err != nil {
							return err
						}
					}
				}

				unwrappedBlk, err := data.Block.PbPhase0UnsignedBlock()
				if err != nil {
					return status.Errorf(codes.Internal,
						"Could not send over stream: %v", err)
				}

				if err := stream.Send(unwrappedBlk); err != nil {
					return status.Errorf(codes.Unavailable,
						"Could not send over stream: %v", err)
				}

				log.WithField("slot", data.Block.Slot()).Debug(
					"Sent block to orchestrator")
			}
		case <-pBlockSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}
