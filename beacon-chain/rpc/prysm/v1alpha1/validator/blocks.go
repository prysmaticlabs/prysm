package validator

import (
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamBlocksAltair to clients every single time a block is received by the beacon node.
func (vs *Server) StreamBlocksAltair(req *ethpb.StreamBlocksRequest, stream ethpb.BeaconNodeValidator_StreamBlocksAltairServer) error {
	blocksChannel := make(chan *feed.Event, 1)
	var blockSub event.Subscription
	if req.VerifiedOnly {
		blockSub = vs.StateNotifier.StateFeed().Subscribe(blocksChannel)
	} else {
		blockSub = vs.BlockNotifier.BlockFeed().Subscribe(blocksChannel)
	}
	defer blockSub.Unsubscribe()

	for {
		select {
		case blockEvent := <-blocksChannel:
			if req.VerifiedOnly {
				if blockEvent.Type == statefeed.BlockProcessed {
					data, ok := blockEvent.Data.(*statefeed.BlockProcessedData)
					if !ok || data == nil {
						continue
					}
					b := &ethpb.StreamBlocksResponse{}
					switch data.SignedBlock.Version() {
					case version.Phase0:
						phBlk, ok := data.SignedBlock.Proto().(*ethpb.SignedBeaconBlock)
						if !ok {
							log.Warn("Mismatch between version and block type, was expecting *ethpb.SignedBeaconBlock")
							continue
						}
						b.Block = &ethpb.StreamBlocksResponse_Phase0Block{Phase0Block: phBlk}
					case version.Altair:
						phBlk, ok := data.SignedBlock.Proto().(*ethpb.SignedBeaconBlockAltair)
						if !ok {
							log.Warn("Mismatch between version and block type, was expecting *v2.SignedBeaconBlockAltair")
							continue
						}
						b.Block = &ethpb.StreamBlocksResponse_AltairBlock{AltairBlock: phBlk}
					}

					if err := stream.Send(b); err != nil {
						return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
					}
				}
			} else {
				if blockEvent.Type == blockfeed.ReceivedBlock {
					data, ok := blockEvent.Data.(*blockfeed.ReceivedBlockData)
					if !ok {
						// Got bad data over the stream.
						continue
					}
					if data.SignedBlock == nil {
						// One nil block shouldn't stop the stream.
						continue
					}
					headState, err := vs.HeadFetcher.HeadState(vs.Ctx)
					if err != nil {
						log.WithError(err).WithField("blockSlot", data.SignedBlock.Block().Slot()).Error("Could not get head state")
						continue
					}
					signed := data.SignedBlock
					if err := blocks.VerifyBlockSignature(headState, signed.Block().ProposerIndex(), signed.Signature(), signed.Block().HashTreeRoot); err != nil {
						log.WithError(err).WithField("blockSlot", data.SignedBlock.Block().Slot()).Error("Could not verify block signature")
						continue
					}
					b := &ethpb.StreamBlocksResponse{}
					switch data.SignedBlock.Version() {
					case version.Phase0:
						phBlk, ok := data.SignedBlock.Proto().(*ethpb.SignedBeaconBlock)
						if !ok {
							log.Warn("Mismatch between version and block type, was expecting *ethpb.SignedBeaconBlock")
							continue
						}
						b.Block = &ethpb.StreamBlocksResponse_Phase0Block{Phase0Block: phBlk}
					case version.Altair:
						phBlk, ok := data.SignedBlock.Proto().(*ethpb.SignedBeaconBlockAltair)
						if !ok {
							log.Warn("Mismatch between version and block type, was expecting *v2.SignedBeaconBlockAltair")
							continue
						}
						b.Block = &ethpb.StreamBlocksResponse_AltairBlock{AltairBlock: phBlk}
					}
					if err := stream.Send(b); err != nil {
						return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
					}
				}
			}
		case <-blockSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-vs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}
