package p2p

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

func (p *Service) AddConnectionHandler(reqFunc func(ctx context.Context, topic string, id peer.ID) error, topic string) {
	p.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				ctx := context.Background()
				log.WithField("peer", conn.RemotePeer()).Debug(
					"Performing handshake with to peer",
				)
				if err := reqFunc(ctx, topic, conn.RemotePeer()); err != nil {
					log.WithError(err).Error("Could not send succesful hello rpc request")
					if err := p.Disconnect(conn.RemotePeer()); err != nil {
						log.WithError(err).Errorf("Unable to close peer %s", conn.RemotePeer())
					}
					return
				}
			}()
		},
	})
}
