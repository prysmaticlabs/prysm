package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// goodbyeRPCHandler reads the incoming goodbye rpc message from the peer.
func (r *RegularSync) goodbyeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m := msg.(*pb.Goodbye)
	log := log.WithField("Reason", m.Reason.String())
	log.Infof("Peer %s has sent a goodbye message", stream.Conn().RemotePeer())
	// closes all streams with the peer
	return r.p2p.Disconnect(stream.Conn().RemotePeer())
}
