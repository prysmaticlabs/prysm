package p2p

import (
	"context"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var handshakes = make(map[peer.ID]*pb.Status)
var handshakeLock sync.Mutex

// AddHandshake to the local records for initial sync.
func (s *Service) AddHandshake(pid peer.ID, hello *pb.Status) {
	handshakeLock.Lock()
	defer handshakeLock.Unlock()
	handshakes[pid] = hello
}

// Handshakes has not been implemented yet and it may be moved to regular sync...
func (s *Service) Handshakes() map[peer.ID]*pb.Status {
	return nil
}

// AddConnectionHandler adds a callback function which handles the connection with a
// newly added peer. It performs a handshake with that peer by sending a hello request
// and validating the response from the peer.
func (s *Service) AddConnectionHandler(reqFunc func(ctx context.Context, id peer.ID) error) {
	s.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				ctx := context.Background()
				log := log.WithField("peer", conn.RemotePeer())
				if conn.Stat().Direction == network.DirInbound {
					log.Debug("Not sending hello for inbound connection")
					return
				}
				log.Debug("Performing handshake with peer")
				if err := reqFunc(ctx, conn.RemotePeer()); err != nil && err != io.EOF {
					log.WithError(err).Error("Could not send successful hello rpc request")
					log.Error("Not disconnecting for interop testing :)")
					//if err := s.Disconnect(conn.RemotePeer()); err != nil {
					//	log.WithError(err).Errorf("Unable to close peer %s", conn.RemotePeer())
					//	return
					//}
					return
				}
				log.WithField("peer", conn.RemotePeer().Pretty()).Info("New peer connected.")
			}()
		},
	})
}

// addDisconnectionHandler ensures that previously disconnected peers aren't dialed again. Due
// to either their ports being closed, nodes are no longer active,etc.
func (s *Service) addDisconnectionHandler() {
	s.host.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				s.exclusionList.Set(conn.RemotePeer().String(), true, ttl)
				log.WithField("peer", conn.RemotePeer()).Debug(
					"Peer is added to exclusion list",
				)
			}()
		},
	})
}
