package sync

import (
	"context"
	"fmt"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
)

const (
	codeClientShutdown uint64 = iota
	codeWrongNetwork
	codeGenericError
)

var goodByes = map[uint64]string{
	codeClientShutdown: "client shutdown",
	codeWrongNetwork:   "irrelevant network",
	codeGenericError:   "fault/error",
}

// goodbyeRPCHandler reads the incoming goodbye rpc message from the peer.
func (r *Service) goodbyeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m, ok := msg.(*uint64)
	if !ok {
		return fmt.Errorf("wrong message type for goodbye, got %T, wanted *uint64", msg)
	}
	log := log.WithField("Reason", goodbyeMessage(*m))
	log.WithField("peer", stream.Conn().RemotePeer()).Info("Peer has sent a goodbye message")
	// closes all streams with the peer
	return r.p2p.Disconnect(stream.Conn().RemotePeer())
}

func goodbyeMessage(num uint64) string {
	reason, ok := goodByes[num]
	if ok {
		return reason
	}
	return fmt.Sprintf("unknown goodbye value of %d Received", num)
}
