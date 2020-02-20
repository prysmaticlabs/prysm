package sync

import (
	"bytes"
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
)

// maintainPeerStatuses by infrequently polling peers for their latest status.
func (r *Service) maintainPeerStatuses() {
	// Run twice per epoch.
	interval := time.Duration(params.BeaconConfig().SecondsPerSlot*params.BeaconConfig().SlotsPerEpoch/2) * time.Second
	runutil.RunEvery(r.ctx, interval, func() {
		for _, pid := range r.p2p.Peers().Connected() {
			go func(id peer.ID) {
				// If the status hasn't been updated in the recent interval time.
				lastUpdated, err := r.p2p.Peers().ChainStateLastUpdated(id)
				if err != nil {
					// Peer has vanished; nothing to do.
					return
				}
				if roughtime.Now().After(lastUpdated.Add(interval)) {
					if err := r.sendRPCStatusRequest(r.ctx, id); err != nil {
						log.WithField("peer", id).WithError(err).Error("Failed to request peer status")
					}
				}
			}(pid)
		}
	})
}

// resyncIfBehind checks periodically to see if we are in normal sync but have fallen behind our peers by more than an epoch,
// in which case we attempt a resync using the initial sync method to catch up.
func (r *Service) resyncIfBehind() {
	// Run sixteen times per epoch.
	interval := time.Duration(params.BeaconConfig().SecondsPerSlot*params.BeaconConfig().SlotsPerEpoch/16) * time.Second
	runutil.RunEvery(r.ctx, interval, func() {
		currentEpoch := uint64(roughtime.Now().Unix()-r.chain.GenesisTime().Unix()) / (params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch)
		syncedEpoch := helpers.SlotToEpoch(r.chain.HeadSlot())
		if r.initialSync != nil && !r.initialSync.Syncing() && syncedEpoch < currentEpoch-1 {
			_, highestEpoch, _ := r.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, syncedEpoch)
			if helpers.StartSlot(highestEpoch) > r.chain.HeadSlot() {
				log.WithFields(logrus.Fields{
					"currentEpoch": currentEpoch,
					"syncedEpoch":  syncedEpoch,
					"peersEpoch":   highestEpoch,
				}).Info("Fallen behind peers; reverting to initial sync to catch up")
				numberOfTimesResyncedCounter.Inc()
				r.clearPendingSlots()
				if err := r.initialSync.Resync(); err != nil {
					log.Errorf("Could not resync chain: %v", err)
				}
			}
		}
	})
}

// sendRPCStatusRequest for a given topic with an expected protobuf message type.
func (r *Service) sendRPCStatusRequest(ctx context.Context, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	headRoot, err := r.chain.HeadRoot(ctx)
	if err != nil {
		return err
	}

	resp := &pb.Status{
		HeadForkVersion: r.chain.CurrentFork().CurrentVersion,
		FinalizedRoot:   r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch:  r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:        headRoot,
		HeadSlot:        r.chain.HeadSlot(),
	}
	stream, err := r.p2p.Send(ctx, resp, id)
	if err != nil {
		return err
	}

	code, errMsg, err := ReadStatusCode(stream, r.p2p.Encoding())
	if err != nil {
		return err
	}

	if code != 0 {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		return errors.New(errMsg)
	}

	msg := &pb.Status{}
	if err := r.p2p.Encoding().DecodeWithLength(stream, msg); err != nil {
		return err
	}
	r.p2p.Peers().SetChainState(stream.Conn().RemotePeer(), msg)

	err = r.validateStatusMessage(msg, stream)
	if err != nil {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
	}
	return err
}

func (r *Service) removeDisconnectedPeerStatus(ctx context.Context, pid peer.ID) error {
	return nil
}

// statusRPCHandler reads the incoming Status RPC from the peer and responds with our version of a status message.
// This handler will disconnect any peer that does not match our fork version.
func (r *Service) statusRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "status")
	m := msg.(*pb.Status)

	if err := r.validateStatusMessage(m, stream); err != nil {
		log.WithField("peer", stream.Conn().RemotePeer()).Debug("Invalid fork version from peer")
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		originalErr := err
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, err.Error())
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				// The peer may already be ignoring us, as we disagree on fork version, so log this as debug only.
				log.WithError(err).Debug("Failed to write to stream")
			}
		}
		stream.Close() // Close before disconnecting.
		// Add a short delay to allow the stream to flush before closing the connection.
		// There is still a chance that the peer won't receive the message.
		time.Sleep(50 * time.Millisecond)
		if err := r.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
			log.WithError(err).Error("Failed to disconnect from peer")
		}
		return originalErr
	}
	r.p2p.Peers().SetChainState(stream.Conn().RemotePeer(), m)

	headRoot, err := r.chain.HeadRoot(ctx)
	if err != nil {
		return err
	}

	resp := &pb.Status{
		HeadForkVersion: r.chain.CurrentFork().CurrentVersion,
		FinalizedRoot:   r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch:  r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:        headRoot,
		HeadSlot:        r.chain.HeadSlot(),
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err = r.p2p.Encoding().EncodeWithLength(stream, resp)

	return err
}

func (r *Service) validateStatusMessage(msg *pb.Status, stream network.Stream) error {
	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, msg.HeadForkVersion) {
		return errWrongForkVersion
	}
	genesis := r.chain.GenesisTime()
	maxEpoch := slotutil.EpochsSinceGenesis(genesis)
	// It would take a minimum of 2 epochs to finalize a
	// previous epoch
	maxFinalizedEpoch := maxEpoch - 2
	if msg.FinalizedEpoch > maxFinalizedEpoch {
		return errInvalidEpoch
	}
	return nil
}
