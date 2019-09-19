package sync

import (
	"bytes"
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// sendRPCStatusRequest for a given topic with an expected protobuf message type.
func (r *RegularSync) sendRPCStatusRequest(ctx context.Context, id peer.ID) error {
	log := log.WithField("rpc", "status")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// return if status already exists
	hello := r.statusTracker[id]
	if hello != nil {
		log.Debugf("Peer %s already exists", id)
		return nil
	}

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
	r.statusTrackerLock.Lock()
	r.statusTracker[stream.Conn().RemotePeer()] = msg
	r.statusTrackerLock.Unlock()

	return r.validateStatusMessage(msg, stream)
}

func (r *RegularSync) removeDisconnectedPeerStatus(ctx context.Context, pid peer.ID) error {
	r.helloTrackerLock.Lock()
	delete(r.helloTracker, pid)
	r.helloTrackerLock.Unlock()
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

	// return if hello already exists
	r.statusTrackerLock.RLock()
	hello := r.statusTracker[stream.Conn().RemotePeer()]
	r.statusTrackerLock.RUnlock()
	if hello != nil {
		log.Debugf("Peer %s already exists", stream.Conn().RemotePeer())
		return nil
	}

	m := msg.(*pb.Status)

	r.statusTrackerLock.Lock()
	r.statusTracker[stream.Conn().RemotePeer()] = m
	r.statusTrackerLock.Unlock()

	if err := r.validateStatusMessage(m, stream); err != nil {
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

	r.statusTrackerLock.Lock()
	r.statusTracker[stream.Conn().RemotePeer()] = m
	r.statusTrackerLock.Unlock()

	r.p2p.AddHandshake(stream.Conn().RemotePeer(), m)

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
