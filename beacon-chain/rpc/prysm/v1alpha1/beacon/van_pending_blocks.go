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
func (bs *Server) StreamNewPendingBlocks(
	request *ethpb.StreamPendingBlocksRequest,
	stream ethpb.BeaconChain_StreamNewPendingBlocksServer,
) error {
	// prepareResponse prepares pending block info response
	prepareBlockInfo := func(block *ethpb.BeaconBlock) (*ethpb.StreamPendingBlockInfo, error) {
		// retrieving finalized epoch and slot
		finalizedCheckpoint := bs.FinalizationFetcher.FinalizedCheckpt()
		fSlot, err := helpers.StartSlot(finalizedCheckpoint.Epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not send over stream: %v", err)
		}

		// sending block info with finalized slot and epoch
		return &ethpb.StreamPendingBlockInfo{
			Block:          block,
			FinalizedSlot:  fSlot,
			FinalizedEpoch: finalizedCheckpoint.Epoch,
		}, nil
	}

	// batchSender sends blocks from specific start epoch to end epoch
	batchSender := func(start, end types.Epoch) error {
		for i := start; i <= end; i++ {
			blks, _, err := bs.BeaconDB.Blocks(bs.Ctx, filters.NewFilter().SetStartEpoch(i).SetEndEpoch(i))
			if err != nil {
				return status.Errorf(codes.Internal, "Could not send batch of previous blocks over stream: %v", err)
			}
			for _, blk := range blks {
				// we do not send block #0 to orchestrator
				if blk.Block().Slot() == 0 {
					continue
				}
				unwrappedBlk, err := blk.PbPhase0Block()
				if err != nil {
					return status.Errorf(codes.Internal, "Could not send over of previous blocks stream: %v", err)
				}

				blockInfo, err := prepareBlockInfo(unwrappedBlk.Block)
				if err != nil {
					return err
				}

				if err := stream.Send(blockInfo); err != nil {
					return status.Errorf(codes.Unavailable, "Could not send over of previous blocks stream: %v", err)
				}
			}
		}
		return nil
	}

	// sender method sends block from specific start slot to end slot
	sender := func(start, end types.Slot) error {
		blks, _, err := bs.BeaconDB.Blocks(bs.Ctx, filters.NewFilter().SetStartSlot(start).SetEndSlot(end))
		if err != nil {
			return err
		}
		for _, blk := range blks {
			// we do not send block #0 to orchestrator
			if blk.Block().Slot() == 0 {
				continue
			}
			unwrappedBlk, err := blk.PbPhase0Block()
			if err != nil {
				return status.Errorf(codes.Internal, "Could not send over stream: %v", err)
			}

			blockInfo, err := prepareBlockInfo(unwrappedBlk.Block)
			if err != nil {
				return err
			}

			if err := stream.Send(blockInfo); err != nil {
				return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
			}
		}
		return nil
	}

	// publishing previous blocks from requested slot to finalized checkpoint
	cp, err := bs.BeaconDB.FinalizedCheckpoint(bs.Ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not retrieve finalize epoch: %v", err)
	}
	epochStart := helpers.SlotToEpoch(request.FromSlot)
	epochEnd := cp.Epoch
	if epochStart <= epochEnd {
		if err := batchSender(epochStart, epochEnd); err != nil {
			return err
		}
		log.WithField("startEpoch", epochStart).WithField("endEpoch", epochEnd).
			Info("Published previous blocks in batch to finalized checkpoint")
	}

	// publishing previous blocks from finalized epoch to head block
	headBlock, err := bs.HeadFetcher.HeadBlock(bs.Ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not send over of previous blocks stream: %v", err)
	}
	if headBlock == nil || headBlock.IsNil() {
		return status.Errorf(codes.Internal, "Could not send over of previous blocks stream: head block is nil")
	}
	startSlot, err := helpers.EndSlot(epochEnd)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not send over of previous blocks stream: %v", err)
	}
	endSlot := headBlock.Block().Slot()
	if startSlot+1 <= endSlot {
		if err := sender(startSlot, endSlot); err != nil {
			return err
		}
		log.WithField("startSlot", startSlot+1).WithField("endSlot", endSlot).
			Info("Published blocks from finalized epoch to head block")
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
				// we are sending new blocks that are added by the time of sending previous block in upper segment of code
				// so we need to send those blocks before sending current block
				if firstTime {
					firstTime = false
					startSlot = endSlot + 1
					endSlot = data.Block.Slot()
					log.WithField("startSlot", startSlot).WithField("endSlot", endSlot).WithField("liveSyncStart", endSlot+1).
						Info("Sending left over blocks")
					if startSlot < endSlot {
						if err := sender(startSlot, endSlot); err != nil {
							return err
						}
					}
				}
				// Unwrapping beacon block
				unwrappedBlk, err := data.Block.PbPhase0UnsignedBlock()
				if err != nil {
					return status.Errorf(codes.Internal, "Could not send over stream: %v", err)
				}

				blockInfo, err := prepareBlockInfo(unwrappedBlk)
				if err != nil {
					return err
				}

				// sending block info with finalized slot and epoch
				if err := stream.Send(blockInfo); err != nil {
					return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
				}
				log.WithField("slot", data.Block.Slot()).Debug("Sent block to orchestrator")
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
