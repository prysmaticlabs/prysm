package beaconclient

import (
	"context"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/sirupsen/logrus"
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

	go bs.collectReceivedAttestations(ctx)
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
		bs.receivedAttestationsBuffer <- res
	}
}

func (bs *Service) collectReceivedAttestations(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.collectReceivedAttestations")
	defer span.End()

	var atts []*ethpb.IndexedAttestation
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			if len(atts) > 0 {
				bs.collectedAttestationsBuffer <- atts
				atts = []*ethpb.IndexedAttestation{}
			}
		case att := <-bs.receivedAttestationsBuffer:
			atts = append(atts, att)
		case collectedAtts := <-bs.collectedAttestationsBuffer:
			if err := bs.slasherDB.SaveIndexedAttestations(ctx, collectedAtts); err != nil {
				log.WithError(err).Error("Could not save indexed attestation")
				continue
			}
			log.Infof("%d attestations for slot %d saved to slasher DB", len(collectedAtts), collectedAtts[0].Data.Slot)
			slasherNumAttestationsReceived.Add(float64(len(collectedAtts)))

			// After saving, we send the received attestation over the attestation feed.
			for _, att := range collectedAtts {
				log.WithFields(logrus.Fields{
					"slot":    att.Data.Slot,
					"indices": att.AttestingIndices,
				}).Debug("Sending attestation to detection service")
				bs.attestationFeed.Send(att)
			}
		case <-ctx.Done():
			return
		}
	}
}
