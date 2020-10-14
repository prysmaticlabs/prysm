package sync

import (
	"context"
	"fmt"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/sirupsen/logrus"
)

const (
	codeClientShutdown types.SSZUint64 = iota
	codeWrongNetwork
	codeGenericError
)

var goodByes = map[types.SSZUint64]string{
	codeClientShutdown: "client shutdown",
	codeWrongNetwork:   "irrelevant network",
	codeGenericError:   "fault/error",
}

// Add a short delay to allow the stream to flush before resetting it.
// There is still a chance that the peer won't receive the message.
const flushDelay = 50 * time.Millisecond

// goodbyeRPCHandler reads the incoming goodbye rpc message from the peer.
func (s *Service) goodbyeRPCHandler(_ context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	SetRPCStreamDeadlines(stream)

	m, ok := msg.(*types.SSZUint64)
	if !ok {
		return fmt.Errorf("wrong message type for goodbye, got %T, wanted *uint64", msg)
	}
	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)
	log := log.WithField("Reason", goodbyeMessage(*m))
	log.WithField("peer", stream.Conn().RemotePeer()).Debug("Peer has sent a goodbye message")
	// closes all streams with the peer
	return s.p2p.Disconnect(stream.Conn().RemotePeer())
}

func (s *Service) sendGoodByeAndDisconnect(ctx context.Context, code types.SSZUint64, id peer.ID) error {
	if err := s.sendGoodByeMessage(ctx, code, id); err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
			"peer":  id,
		}).Debug("Could not send goodbye message to peer")
	}
	return s.p2p.Disconnect(id)
}

func (s *Service) sendGoodByeMessage(ctx context.Context, code types.SSZUint64, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	stream, err := s.p2p.Send(ctx, &code, p2p.RPCGoodByeTopic, id)
	if err != nil {
		return err
	}
	defer func() {
		if err := helpers.FullClose(stream); err != nil && err.Error() != mux.ErrReset.Error() {
			log.WithError(err).Debugf("Failed to reset stream with protocol %s", stream.Protocol())
		}
	}()
	log := log.WithField("Reason", goodbyeMessage(code))
	log.WithField("peer", stream.Conn().RemotePeer()).Debug("Sending Goodbye message to peer")
	return nil
}

func goodbyeMessage(num types.SSZUint64) string {
	reason, ok := goodByes[num]
	if ok {
		return reason
	}
	return fmt.Sprintf("unknown goodbye value of %d Received", num)
}
