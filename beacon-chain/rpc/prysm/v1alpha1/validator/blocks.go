package validator

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
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
				if err := sendVerifiedBlocks(stream, blockEvent); err != nil {
					return err
				}
			} else {
				if err := vs.sendBlocks(stream, blockEvent); err != nil {
					return err
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

func sendVerifiedBlocks(stream ethpb.BeaconNodeValidator_StreamBlocksAltairServer, blockEvent *feed.Event) error {
	if blockEvent.Type != statefeed.BlockProcessed {
		return nil
	}
	data, ok := blockEvent.Data.(*statefeed.BlockProcessedData)
	if !ok || data == nil {
		return nil
	}
	b := &ethpb.StreamBlocksResponse{}
	switch data.SignedBlock.Version() {
	case version.Phase0:
		pb, err := data.SignedBlock.Proto()
		if err != nil {
			return errors.Wrap(err, "could not get protobuf block")
		}
		phBlk, ok := pb.(*ethpb.SignedBeaconBlock)
		if !ok {
			log.Warn("Mismatch between version and block type, was expecting SignedBeaconBlock")
			return nil
		}
		b.Block = &ethpb.StreamBlocksResponse_Phase0Block{Phase0Block: phBlk}
	case version.Altair:
		pb, err := data.SignedBlock.Proto()
		if err != nil {
			return errors.Wrap(err, "could not get protobuf block")
		}
		phBlk, ok := pb.(*ethpb.SignedBeaconBlockAltair)
		if !ok {
			log.Warn("Mismatch between version and block type, was expecting SignedBeaconBlockAltair")
			return nil
		}
		b.Block = &ethpb.StreamBlocksResponse_AltairBlock{AltairBlock: phBlk}
	case version.Bellatrix:
		pb, err := data.SignedBlock.Proto()
		if err != nil {
			return errors.Wrap(err, "could not get protobuf block")
		}
		phBlk, ok := pb.(*ethpb.SignedBeaconBlockBellatrix)
		if !ok {
			log.Warn("Mismatch between version and block type, was expecting SignedBeaconBlockBellatrix")
			return nil
		}
		b.Block = &ethpb.StreamBlocksResponse_BellatrixBlock{BellatrixBlock: phBlk}
	}

	if err := stream.Send(b); err != nil {
		return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
	}

	return nil
}

func (vs *Server) sendBlocks(stream ethpb.BeaconNodeValidator_StreamBlocksAltairServer, blockEvent *feed.Event) error {
	if blockEvent.Type != blockfeed.ReceivedBlock {
		return nil
	}

	data, ok := blockEvent.Data.(*blockfeed.ReceivedBlockData)
	if !ok || data == nil {
		// Got bad data over the stream.
		return nil
	}
	if data.SignedBlock == nil {
		// One nil block shouldn't stop the stream.
		return nil
	}
	log := log.WithField("blockSlot", data.SignedBlock.Block().Slot())
	headState, err := vs.HeadFetcher.HeadState(vs.Ctx)
	if err != nil {
		log.WithError(err).Error("Could not get head state")
		return nil
	}
	signed := data.SignedBlock
	if err := blocks.VerifyBlockSignature(headState, signed.Block().ProposerIndex(), signed.Signature(), signed.Block().HashTreeRoot); err != nil {
		log.WithError(err).Error("Could not verify block signature")
		return nil
	}
	b := &ethpb.StreamBlocksResponse{}
	pb, err := data.SignedBlock.Proto()
	if err != nil {
		return errors.Wrap(err, "could not get protobuf block")
	}
	switch p := pb.(type) {
	case *ethpb.SignedBeaconBlock:
		b.Block = &ethpb.StreamBlocksResponse_Phase0Block{Phase0Block: p}
	case *ethpb.SignedBeaconBlockAltair:
		b.Block = &ethpb.StreamBlocksResponse_AltairBlock{AltairBlock: p}
	case *ethpb.SignedBeaconBlockBellatrix:
		b.Block = &ethpb.StreamBlocksResponse_BellatrixBlock{BellatrixBlock: p}
	default:
		log.Errorf("Unknown block type %T", p)
	}
	if err := stream.Send(b); err != nil {
		return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
	}

	return nil
}
