package sync

import (
	"bytes"
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

const statusInterval = 6 * time.Minute // 60 slots.

// maintainPeerStatuses by infrequently polling peers for their latest status.
func (r *RegularSync) maintainPeerStatuses() {
	ticker := time.NewTicker(statusInterval)
	for {
		ctx := context.Background()
		select {
		case <-ticker.C:
			for _, pid := range peerstatus.Keys() {
				// If the status hasn't been updated in the recent interval time.
				if roughtime.Now().After(peerstatus.LastUpdated(pid).Add(statusInterval)) {
					if err := r.sendRPCStatusRequest(ctx, pid); err != nil {
						log.WithError(err).Error("Failed to request peer status")
					}
				}
			}

		case <-r.ctx.Done():
			return
		}
	}
}

// sendRPCStatusRequest for a given topic with an expected protobuf message type.
func (r *RegularSync) sendRPCStatusRequest(ctx context.Context, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp := &pb.Status{
		HeadForkVersion: params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:   r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch:  r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:        r.chain.HeadRoot(),
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
		return errors.New(errMsg)
	}

	msg := &pb.Status{}
	if err := r.p2p.Encoding().DecodeWithLength(stream, msg); err != nil {
		return err
	}
	peerstatus.Set(stream.Conn().RemotePeer(), msg)

	err = r.validateStatusMessage(msg, stream)
	if err != nil {
		peerstatus.IncreaseFailureCount(stream.Conn().RemotePeer())
	}
	return err
}

func (r *RegularSync) removeDisconnectedPeerStatus(ctx context.Context, pid peer.ID) error {
	peerstatus.Delete(pid)
	return nil
}

// statusRPCHandler reads the incoming Status RPC from the peer and responds with our version of a status message.
// This handler will disconnect any peer that does not match our fork version.
func (r *RegularSync) statusRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("handler", "status")
	m := msg.(*pb.Status)

	peerstatus.Set(stream.Conn().RemotePeer(), m)

	if err := r.validateStatusMessage(m, stream); err != nil {
		peerstatus.IncreaseFailureCount(stream.Conn().RemotePeer())
		originalErr := err
		resp, err := r.generateErrorResponse(responseCodeInvalidRequest, err.Error())
		if err != nil {
			log.WithError(err).Error("Failed to generate a response error")
		} else {
			if _, err := stream.Write(resp); err != nil {
				log.WithError(err).Errorf("Failed to write to stream")
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

	resp := &pb.Status{
		HeadForkVersion: params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:   r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch:  r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:        r.chain.HeadRoot(),
		HeadSlot:        r.chain.HeadSlot(),
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err := r.p2p.Encoding().EncodeWithLength(stream, resp)

	return err
}

func (r *RegularSync) validateStatusMessage(msg *pb.Status, stream network.Stream) error {
	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, msg.HeadForkVersion) {
		return errWrongForkVersion
	}
	return nil
}
