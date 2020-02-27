package beaconclient

import (
	"context"
	"io"

	ptypes "github.com/gogo/protobuf/types"
	"go.opencensus.io/trace"
)

// receiveBlocks starts a gRPC client stream listener to obtain
// blocks from the beacon node. Upon receiving a block, the service
// broadcasts it to a feed for other services in slasher to subscribe to.
func (bs *Service) receiveBlocks(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.receiveBlocks")
	defer span.End()
	stream, err := bs.beaconClient.StreamBlocks(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks stream")
		return
	}
	for {
		res, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			log.WithError(ctx.Err()).Error("Context canceled - shutting down blocks receiver")
			return
		}
		if err != nil {
			log.WithError(err).Error("Could not receive block from beacon node")
		}
		log.WithField("slot", res.Block.Slot).Debug("Received block from beacon node")
		// We send the received block over the block feed.
		bs.blockFeed.Send(res)
	}
}

// receiveAttestations starts a gRPC client stream listener to obtain
// attestations from the beacon node. Upon receiving an attestation, the service
// broadcasts it to a feed for other services in slasher to subscribe to.
func (bs *Service) receiveAttestations(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.receiveAttestations")
	defer span.End()
	stream, err := bs.beaconClient.StreamIndexedAttestations(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Failed to retrieve attestations stream")
		return
	}
	for {
		res, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			log.WithError(ctx.Err()).Error("Context canceled - shutting down attestations receiver")
			return
		}
		if err != nil {
			log.WithError(err).Error("Could not receive attestations from beacon node")
			return
		}
		log.WithField("slot", res.Data.Slot).Debug("Received attestation from beacon node")
		if err := bs.slasherDB.SaveIncomingIndexedAttestationByEpoch(ctx, res); err != nil {
			log.WithError(err).Error("Could not save indexed attestation")
			continue
		}
		// We send the received attestation over the attestation feed.
		bs.attestationFeed.Send(res)
	}
}
