package sync

import (
	"context"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
)

var goodByes = map[uint64]string{
	0: "Client Shut Down",
	1: "Irrelevant Network",
	2: "Fault/Error",
}

func (r *RegularSync) transformGoodbyeVal(num uint64) string {
	reason, ok := goodByes[num]
	if ok {
		return reason
	}
	return "Unknown Goodbye Value Received"
}

// goodbyeRPCHandler reads the incoming goodbye rpc message from the peer.
func (r *RegularSync) goodbyeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m := msg.(uint64)
	log := log.WithField("Reason", r.transformGoodbyeVal(m))
	log.Infof("Peer %s has sent a goodbye message", stream.Conn().RemotePeer())
	// closes all streams with the peer
	return r.p2p.Disconnect(stream.Conn().RemotePeer())
}
