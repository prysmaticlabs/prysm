package beacon

import (
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/shared/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO- Need to add from slot number for getting previous blocks from chain
// StreamNewPendingBlocks to orchestrator client every single time an unconfirmed block is received by the beacon node.
func (bs *Server) StreamNewPendingBlocks(empty *ptypes.Empty, stream ethpb.BeaconChain_StreamNewPendingBlocksServer) error {
	unConfirmedblocksChannel := make(chan *feed.Event, 1)
	var unConfirmedblockSub event.Subscription

	unConfirmedblockSub = bs.BlockNotifier.BlockFeed().Subscribe(unConfirmedblocksChannel)
	defer unConfirmedblockSub.Unsubscribe()

	// Getting un-confirmed blocks from cache and sends those blocks to orchestrator
	unconfirmedBlocks, err := bs.UnconfirmedBlockFetcher.SortedUnConfirmedBlocksFromCache()
	if err != nil {
		return status.Errorf(codes.Unavailable, "Could not send cached un-confirmed blocks over stream: %v", err)
	}
	for _, blk := range unconfirmedBlocks {
		if err := stream.Send(blk); err != nil {
			return status.Errorf(codes.Unavailable, "Could not send un-confirmed block over stream: %v", err)
		}
	}

	for {
		select {
		case blockEvent := <-unConfirmedblocksChannel:
			if blockEvent.Type == blockfeed.UnConfirmedBlock {
				data, ok := blockEvent.Data.(*blockfeed.UnConfirmedBlockData)
				if !ok || data == nil {
					continue
				}
				if err := stream.Send(data.Block); err != nil {
					return status.Errorf(codes.Unavailable, "Could not send un-confirmed block over stream: %v", err)
				}
				log.WithField("slot", data.Block.Slot).Debug(
					"New pending block has been published successfully")
			}
		case <-unConfirmedblockSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}
