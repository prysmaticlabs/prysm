package p2p

import (
	"context"
	"sync"

	"github.com/gogo/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	syncHandler "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var handshakes = make(map[peer.ID]*pb.Hello)
var handshakeLock sync.Mutex

// AddHandshake to the local records for initial sync.
func (p *Service) AddHandshake(pid peer.ID, hello *pb.Hello) {
	handshakeLock.Lock()
	defer handshakeLock.Unlock()
	handshakes[pid] = hello
}

// Handshakes has not been implemented yet and it may be moved to regular sync...
func (p *Service) Handshakes() map[peer.ID]*pb.Hello {
	return nil
}

func connectionHandler(h host.Host, handler syncHandler.RpcHandler, topic string) {
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				ctx := context.Background()
				log.WithField("peer", conn.RemotePeer()).Debug(
					"Performing handshake with to peer",
				)

				s, err := h.NewStream(
					ctx,
					conn.RemotePeer(),
					core.ProtocolID(topic),
				)
				if err != nil {
					log.WithError(err).WithFields(logrus.Fields{
						"peer":    conn.RemotePeer(),
						"address": conn.RemoteMultiaddr(),
					}).Debug("Failed to open stream with newly connected peer")

					if err := h.Network().ClosePeer(conn.RemotePeer()); err != nil {
						log.WithError(err).Errorf("Unable to close peer %s", conn.RemotePeer())
					}
					return
				}
				defer s.Close()
				if err := handler(ctx, proto.Message(nil), s); err != nil {
					log.WithError(err).Error("Could not send hello rpc request")
				}
			}()
		},
	})
}
