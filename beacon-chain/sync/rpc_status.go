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
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
					if err := r.reValidatePeer(r.ctx, id); err != nil {
						log.WithField("peer", id).WithError(err).Error("Failed to revalidate peer")
					}
				}
			}(pid)
		}
	})
}

// resyncIfBehind checks periodically to see if we are in normal sync but have fallen behind our peers by more than an epoch,
// in which case we attempt a resync using the initial sync method to catch up.
func (r *Service) resyncIfBehind() {
	millisecondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch * 1000
	// Run sixteen times per epoch.
	interval := time.Duration(int64(millisecondsPerEpoch)/16) * time.Millisecond
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

	forkDigest, err := r.forkDigest()
	if err != nil {
		return err
	}
	resp := &pb.Status{
		ForkDigest:     forkDigest[:],
		FinalizedRoot:  r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch: r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:       headRoot,
		HeadSlot:       r.chain.HeadSlot(),
	}
	stream, err := r.p2p.Send(ctx, resp, p2p.RPCStatusTopic, id)
	if err != nil {
		return err
	}
	defer func() {
		if err := stream.Reset(); err != nil {
			log.WithError(err).Errorf("Failed to reset stream with protocol %s", stream.Protocol())
		}
	}()

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

	err = r.validateStatusMessage(ctx, msg)
	if err != nil {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		// Disconnect if on a wrong fork.
		if err == errWrongForkDigestVersion {
			if err := r.sendGoodByeAndDisconnect(ctx, codeWrongNetwork, stream.Conn().RemotePeer()); err != nil {
				return err
			}
		}
	}
	return err
}

func (r *Service) reValidatePeer(ctx context.Context, id peer.ID) error {
	if err := r.sendRPCStatusRequest(ctx, id); err != nil {
		return err
	}
	// Do not return an error for ping requests.
	if err := r.sendPingRequest(ctx, id); err != nil {
		log.WithError(err).Debug("Could not ping peer")
	}
	return nil
}

func (r *Service) removeDisconnectedPeerStatus(ctx context.Context, pid peer.ID) error {
	return nil
}

// statusRPCHandler reads the incoming Status RPC from the peer and responds with our version of a status message.
// This handler will disconnect any peer that does not match our fork version.
func (r *Service) statusRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "status")
	m, ok := msg.(*pb.Status)
	if !ok {
		return errors.New("message is not type *pb.Status")
	}

	if err := r.validateStatusMessage(ctx, m); err != nil {
		log.WithFields(logrus.Fields{
			"peer":  stream.Conn().RemotePeer(),
			"error": err}).Debug("Invalid status message from peer")

		respCode := byte(0)
		switch err {
		case errGeneric:
			respCode = responseCodeServerError
		case errWrongForkDigestVersion:
			// Respond with our status and disconnect with the peer.
			r.p2p.Peers().SetChainState(stream.Conn().RemotePeer(), m)
			if err := r.respondWithStatus(ctx, stream); err != nil {
				return err
			}
			if err := stream.Close(); err != nil { // Close before disconnecting.
				log.WithError(err).Error("Failed to close stream")
			}
			if err := r.sendGoodByeAndDisconnect(ctx, codeWrongNetwork, stream.Conn().RemotePeer()); err != nil {
				return err
			}
			return nil
		default:
			respCode = responseCodeInvalidRequest
			r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		}

		originalErr := err
		resp, err := r.generateErrorResponse(respCode, err.Error())
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				// The peer may already be ignoring us, as we disagree on fork version, so log this as debug only.
				log.WithError(err).Debug("Failed to write to stream")
			}
		}
		if err := stream.Close(); err != nil { // Close before disconnecting.
			log.WithError(err).Error("Failed to close stream")
		}
		// Add a short delay to allow the stream to flush before closing the connection.
		// There is still a chance that the peer won't receive the message.
		time.Sleep(50 * time.Millisecond)
		if err := r.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
			log.WithError(err).Error("Failed to disconnect from peer")
		}
		return originalErr
	}
	r.p2p.Peers().SetChainState(stream.Conn().RemotePeer(), m)

	return r.respondWithStatus(ctx, stream)
}

func (r *Service) respondWithStatus(ctx context.Context, stream network.Stream) error {
	headRoot, err := r.chain.HeadRoot(ctx)
	if err != nil {
		return err
	}

	forkDigest, err := r.forkDigest()
	if err != nil {
		return err
	}
	resp := &pb.Status{
		ForkDigest:     forkDigest[:],
		FinalizedRoot:  r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch: r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:       headRoot,
		HeadSlot:       r.chain.HeadSlot(),
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err = r.p2p.Encoding().EncodeWithLength(stream, resp)
	return err
}

func (r *Service) validateStatusMessage(ctx context.Context, msg *pb.Status) error {
	forkDigest, err := r.forkDigest()
	if err != nil {
		return err
	}
	if !bytes.Equal(forkDigest[:], msg.ForkDigest) {
		return errWrongForkDigestVersion
	}
	genesis := r.chain.GenesisTime()
	finalizedEpoch := r.chain.FinalizedCheckpt().Epoch
	maxEpoch := slotutil.EpochsSinceGenesis(genesis)
	// It would take a minimum of 2 epochs to finalize a
	// previous epoch
	maxFinalizedEpoch := uint64(0)
	if maxEpoch > 2 {
		maxFinalizedEpoch = maxEpoch - 2
	}
	if msg.FinalizedEpoch > maxFinalizedEpoch {
		return errInvalidEpoch
	}
	// Exit early if the peer's finalized epoch
	// is less than that of the remote peer's.
	if finalizedEpoch < msg.FinalizedEpoch {
		return nil
	}
	finalizedAtGenesis := msg.FinalizedEpoch == 0
	rootIsEqual := bytes.Equal(params.BeaconConfig().ZeroHash[:], msg.FinalizedRoot)
	// If peer is at genesis with the correct genesis root hash we exit.
	if finalizedAtGenesis && rootIsEqual {
		return nil
	}
	if !r.db.IsFinalizedBlock(context.Background(), bytesutil.ToBytes32(msg.FinalizedRoot)) {
		return errInvalidFinalizedRoot
	}
	blk, err := r.db.Block(ctx, bytesutil.ToBytes32(msg.FinalizedRoot))
	if err != nil {
		return errGeneric
	}
	if blk == nil {
		return errGeneric
	}
	// TODO(#5827) Verify the finalized block with the epoch in the
	// status message
	return nil
}
