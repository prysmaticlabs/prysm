package beaconclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
)

// reconnectPeriod is the frequency that we try to restart our
// streams when the beacon chain is node does not respond.
var reconnectPeriod = 5 * time.Second

// ReceiveBlocks starts a gRPC client stream listener to obtain
// blocks from the beacon node. Upon receiving a block, the service
// broadcasts it to a feed for other services in slasher to subscribe to.
func (bs *Service) ReceiveBlocks(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.ReceiveBlocks")
	defer span.End()
	stream, err := bs.beaconClient.StreamBlocks(ctx, &ethpb.StreamBlocksRequest{} /* Prefers unverified block to catch slashing */)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve blocks stream")
		return
	}
	for {
		res, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if errors.Is(err, io.EOF) {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			log.WithError(ctx.Err()).Error("Context canceled - shutting down blocks receiver")
			return
		}
		if err != nil {
			if e, ok := status.FromError(err); ok {
				switch e.Code() {
				case codes.Canceled, codes.Internal, codes.Unavailable:
					log.WithError(err).Infof("Trying to restart connection. rpc status: %v", e.Code())
					err = bs.restartBeaconConnection(ctx)
					if err != nil {
						log.WithError(err).Error("Could not restart beacon connection")
						return
					}
					stream, err = bs.beaconClient.StreamBlocks(ctx, &ethpb.StreamBlocksRequest{} /* Prefers unverified block to catch slashing */)
					if err != nil {
						log.WithError(err).Error("Could not restart block stream")
						return
					}
					log.Info("Block stream restarted...")
				default:
					log.WithError(err).Errorf("Could not receive block from beacon node. rpc status: %v", e.Code())
					return
				}
			} else {
				log.WithError(err).Error("Could not receive blocks from beacon node")
				return
			}
		}
		if res == nil {
			continue
		}
		root, err := res.Block.HashTreeRoot()
		if err != nil {
			log.WithError(err).Error("Could not hash block")
			return
		}

		log.WithFields(logrus.Fields{
			"slot":           res.Block.Slot,
			"proposer_index": res.Block.ProposerIndex,
			"root":           fmt.Sprintf("%#x...", root[:8]),
		}).Info("Received block from beacon node")
		// We send the received block over the block feed.
		bs.blockFeed.Send(res)
	}
}

// ReceiveAttestations starts a gRPC client stream listener to obtain
// attestations from the beacon node. Upon receiving an attestation, the service
// broadcasts it to a feed for other services in slasher to subscribe to.
func (bs *Service) ReceiveAttestations(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.ReceiveAttestations")
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
		if errors.Is(err, io.EOF) {
			log.Info("Attestation stream closed")
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			log.WithError(ctx.Err()).Error("Context canceled - shutting down attestations receiver")
			return
		}
		if err != nil {
			if e, ok := status.FromError(err); ok {
				switch e.Code() {
				case codes.Canceled, codes.Internal, codes.Unavailable:
					log.WithError(err).Infof("Trying to restart connection. rpc status: %v", e.Code())
					err = bs.restartBeaconConnection(ctx)
					if err != nil {
						log.WithError(err).Error("Could not restart beacon connection")
						return
					}
					stream, err = bs.beaconClient.StreamIndexedAttestations(ctx, &ptypes.Empty{})
					if err != nil {
						log.WithError(err).Error("Could not restart attestation stream")
						return
					}
					log.Info("Attestation stream restarted...")
				default:
					log.WithError(err).Errorf("Could not receive attestations from beacon node. rpc status: %v", e.Code())
					return
				}
			} else {
				log.WithError(err).Error("Could not receive attestations from beacon node")
				return
			}
		}
		if res == nil {
			continue
		}
		bs.receivedAttestationsBuffer <- res
	}
}

func (bs *Service) collectReceivedAttestations(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.collectReceivedAttestations")
	defer span.End()

	var atts []*ethpb.IndexedAttestation
	halfSlot := slotutil.DivideSlotBy(2 /* 1/2 slot duration */)
	ticker := time.NewTicker(halfSlot)
	defer ticker.Stop()
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
			log.WithFields(logrus.Fields{
				"amountSaved": len(collectedAtts),
				"slot":        collectedAtts[0].Data.Slot,
			}).Info("Attestations saved to slasher DB")
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

func (bs *Service) restartBeaconConnection(ctx context.Context) error {
	ticker := time.NewTicker(reconnectPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if bs.conn.GetState() == connectivity.TransientFailure || bs.conn.GetState() == connectivity.Idle {
				log.Debugf("Connection status %v", bs.conn.GetState())
				log.Info("Beacon node is still down")
				continue
			}
			status, err := bs.nodeClient.GetSyncStatus(ctx, &ptypes.Empty{})
			if err != nil {
				log.WithError(err).Error("Could not fetch sync status")
				continue
			}
			if status == nil || status.Syncing {
				log.Info("Waiting for beacon node to be fully synced...")
				continue
			}
			log.Info("Beacon node is fully synced")
			return nil
		case <-ctx.Done():
			log.Debug("Context closed, exiting reconnect routine")
			return errors.New("context closed, no longer attempting to restart stream")
		}
	}
}
