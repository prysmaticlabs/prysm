package beacon

import (
	"context"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// GetCanonicalBlock returns
func (bs *Server) GetCanonicalBlock(ctx context.Context, empty *ptypes.Empty) (*ethpb.SignedBeaconBlock, error) {
	headBlock, err := bs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head block")
	}
	if headBlock == nil || headBlock.Block == nil {
		return nil, status.Error(codes.Internal, "Head block of chain was nil")
	}

	isGenesis := func(cp *ethpb.Checkpoint) bool {
		return bytesutil.ToBytes32(cp.Root) == params.BeaconConfig().ZeroHash && cp.Epoch == 0
	}
	// Retrieve genesis block in the event we have genesis checkpoints.
	genBlock, err := bs.BeaconDB.GenesisBlock(ctx)
	if err != nil || genBlock == nil || genBlock.Block == nil {
		return nil, status.Error(codes.Internal, "Could not get genesis block")
	}

	finalizedCheckpoint := bs.FinalizationFetcher.FinalizedCheckpt()
	if !isGenesis(finalizedCheckpoint) {
		b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(finalizedCheckpoint.Root))
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get finalized block")
		}
		if err := helpers.VerifyNilBeaconBlock(b); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get finalized block: %v", err)
		}
	}

	justifiedCheckpoint := bs.FinalizationFetcher.CurrentJustifiedCheckpt()
	if !isGenesis(justifiedCheckpoint) {
		b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(justifiedCheckpoint.Root))
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get justified block")
		}
		if err := helpers.VerifyNilBeaconBlock(b); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get justified block: %v", err)
		}
	}

	prevJustifiedCheckpoint := bs.FinalizationFetcher.PreviousJustifiedCheckpt()
	if !isGenesis(prevJustifiedCheckpoint) {
		b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(prevJustifiedCheckpoint.Root))
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get prev justified block")
		}
		if err := helpers.VerifyNilBeaconBlock(b); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get prev justified block: %v", err)
		}
	}

	return headBlock, nil
}
