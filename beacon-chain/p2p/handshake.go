package p2p

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
)

// AddConnectionHandler adds a callback function which handles the connection with a
// newly added peer. It performs a handshake with that peer by sending a hello request
// and validating the response from the peer.
func (s *Service) AddConnectionHandler(reqFunc func(ctx context.Context, id peer.ID) error) {
	s.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				if peerstatus.IsBadPeer(conn.RemotePeer()) {
					// Add Peer to gossipsub blacklist
					s.pubsub.BlacklistPeer(conn.RemotePeer())
					log.WithField("peerID", conn.RemotePeer().Pretty()).Debug("Disconnecting with bad peer")
					if err := s.Disconnect(conn.RemotePeer()); err != nil {
						log.WithError(err).Errorf("Unable to close peer %s", conn.RemotePeer())
						return
					}
					return
				}
				ctx := context.Background()
				log := log.WithField("peer", conn.RemotePeer())
				if conn.Stat().Direction == network.DirInbound {
					log.Debug("Not sending hello for inbound connection")
					return
				}
				log.Debug("Performing handshake with peer")
				if err := reqFunc(ctx, conn.RemotePeer()); err != nil && err != io.EOF {
					log.WithError(err).Debug("Could not send successful hello rpc request")
					if err.Error() == "protocol not supported" {
						log.Debug("Not disconnecting peer with unsupported protocol. This may be the DHT node or relay.")
						return
					}
					if err := s.Disconnect(conn.RemotePeer()); err != nil {
						log.WithError(err).Errorf("Unable to close peer %s", conn.RemotePeer())
						return
					}
					return
				}
				log.WithField("peer", conn.RemotePeer().Pretty()).Info("New peer connected")
			}()
		},
	})
}

// AddDisconnectionHandler ensures that previously disconnected peers aren't dialed again. Due
// to either their ports being closed, nodes are no longer active,etc. This also calls the handler
// responsible for maintaining other parts of the sync or p2p system.
func (s *Service) AddDisconnectionHandler(handler func(ctx context.Context, id peer.ID) error) {
	s.host.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				s.exclusionList.Set(conn.RemotePeer().String(), true, ttl)
				log := log.WithField("peer", conn.RemotePeer())
				log.Debug("Peer is added to exclusion list")
				ctx := context.Background()
				if err := handler(ctx, conn.RemotePeer()); err != nil {
					log.WithError(err).Error("Failed to handle disconnecting peer")
				}
			}()

		},
	})
}
