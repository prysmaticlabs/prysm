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

// sendRPCHelloRequest for a given topic with an expected protobuf message type.
func (r *RegularSync) sendRPCHelloRequest(ctx context.Context, id peer.ID) error {
	log := log.WithField("rpc", "hello")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// return if hello already exists
	hello := r.helloTracker[id]
	if hello != nil {
		log.Debugf("Peer %s already exists", id)
		return nil
	}

	resp := &pb.Hello{
		ForkVersion:    params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:  r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch: r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:       r.chain.HeadRoot(),
		HeadSlot:       r.chain.HeadSlot(),
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

	msg := &pb.Hello{}
	if err := r.p2p.Encoding().DecodeWithLength(stream, msg); err != nil {
		return err
	}
	r.helloTrackerLock.Lock()
	r.helloTracker[stream.Conn().RemotePeer()] = msg
	r.helloTrackerLock.Unlock()

	return r.validateHelloMessage(msg, stream)
}

func (r *RegularSync) removeDisconnectedPeerStatus(ctx context.Context, pid peer.ID) error {
	r.helloTrackerLock.Lock()
	delete(r.helloTracker, pid)
	r.helloTrackerLock.Unlock()
	return nil
}

// helloRPCHandler reads the incoming Hello RPC from the peer and responds with our version of a hello message.
// This handler will disconnect any peer that does not match our fork version.
func (r *RegularSync) helloRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)
	log := log.WithField("rpc", "hello")

	// return if hello already exists
	r.helloTrackerLock.RLock()
	hello := r.helloTracker[stream.Conn().RemotePeer()]
	r.helloTrackerLock.RUnlock()
	if hello != nil {
		log.Debugf("Peer %s already exists", stream.Conn().RemotePeer())
		return nil
	}

	m := msg.(*pb.Hello)

	r.helloTrackerLock.Lock()
	r.helloTracker[stream.Conn().RemotePeer()] = m
	r.helloTrackerLock.Unlock()

	if err := r.validateHelloMessage(m, stream); err != nil {
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

	r.helloTrackerLock.Lock()
	r.helloTracker[stream.Conn().RemotePeer()] = m
	r.helloTrackerLock.Unlock()

	r.p2p.AddHandshake(stream.Conn().RemotePeer(), m)

	resp := &pb.Hello{
		ForkVersion:    params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:  r.chain.FinalizedCheckpt().Root,
		FinalizedEpoch: r.chain.FinalizedCheckpt().Epoch,
		HeadRoot:       r.chain.HeadRoot(),
		HeadSlot:       r.chain.HeadSlot(),
	}

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		log.WithError(err).Error("Failed to write to stream")
	}
	_, err := r.p2p.Encoding().EncodeWithLength(stream, resp)

	return err
}

func (r *RegularSync) validateHelloMessage(msg *pb.Hello, stream network.Stream) error {
	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, msg.ForkVersion) {
		return errWrongForkVersion
	}
	return nil
}
