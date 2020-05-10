package p2p

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
)

// AddConnectionHandler adds a callback function which handles the connection with a
// newly added peer. It performs a handshake with that peer by sending a hello request
// and validating the response from the peer.
func (s *Service) AddConnectionHandler(reqFunc func(ctx context.Context, id peer.ID) error,
	goodbyeFunc func(ctx context.Context, id peer.ID) error) {

	peerMap := make(map[peer.ID]bool)
	peerLock := new(sync.Mutex)
	peerHandshaking := func(id peer.ID) bool {
		peerLock.Lock()
		defer peerLock.Unlock()

		if peerMap[id] {
			return true
		}

		peerMap[id] = true
		return false
	}

	peerFinished := func(id peer.ID) {
		peerLock.Lock()
		defer peerLock.Unlock()

		delete(peerMap, id)
	}

	s.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			if peerHandshaking(conn.RemotePeer()) {
				return
			}
			defer peerFinished(conn.RemotePeer())

			log := log.WithField("peer", conn.RemotePeer().Pretty())

			// Handle the various pre-existing conditions that will result in us not handshaking.
			peerConnectionState, err := s.peers.ConnectionState(conn.RemotePeer())
			if err == nil && (peerConnectionState == peers.PeerConnected || peerConnectionState == peers.PeerConnecting) {
				log.WithField("currentState", peerConnectionState).WithField("reason", "already active").Trace("Ignoring connection request")
				return
			}
			s.peers.Add(nil /* ENR */, conn.RemotePeer(), conn.RemoteMultiaddr(), conn.Stat().Direction)
			if len(s.peers.Active()) >= int(s.cfg.MaxPeers) {
				go func() {
					log.WithField("reason", "at peer limit").Trace("Ignoring connection request")
					if err := goodbyeFunc(context.Background(), conn.RemotePeer()); err != nil {
						log.WithError(err).Trace("Unable to send goodbye message to peer")
					}
					if err := s.Disconnect(conn.RemotePeer()); err != nil {
						log.WithError(err).Error("Unable to disconnect from peer")
					}
				}()
				return
			}
			if s.peers.IsBad(conn.RemotePeer()) {
				log.WithField("reason", "bad peer").Trace("Ignoring connection request")
				if err := s.Disconnect(conn.RemotePeer()); err != nil {
					log.WithError(err).Error("Unable to disconnect from peer")
				}
				return
			}

			// Connection handler must be non-blocking as part of libp2p design.
			go func() {
				// Do not perform handshake on inbound dials.
				if conn.Stat().Direction == network.DirInbound {
					_, err := s.peers.ChainState(conn.RemotePeer())
					peerExists := err == nil
					currentTime := roughtime.Now()

					// wait for peer to initiate handshake
					time.Sleep(10 * time.Second)

					disconnectFromPeer := func() {
						s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnecting)
						if err := s.Disconnect(conn.RemotePeer()); err != nil {
							log.WithError(err).Error("Unable to disconnect from peer")
						}
						s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
					}

					// if peer hasn't sent a status request, we disconnect with them
					if _, err := s.peers.ChainState(conn.RemotePeer()); err == peers.ErrPeerUnknown {
						disconnectFromPeer()
						return
					}
					if peerExists {
						updated, err := s.peers.ChainStateLastUpdated(conn.RemotePeer())
						if err != nil {
							disconnectFromPeer()
							return
						}
						if !updated.After(currentTime) {
							disconnectFromPeer()
							return
						}
					}
					s.host.ConnManager().Protect(conn.RemotePeer(), "protocol")
					s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnected)
					return
				}

				// Go through the handshake process.
				multiAddr := fmt.Sprintf("%s/p2p/%s", conn.RemoteMultiaddr().String(), conn.RemotePeer().String())
				log := log.WithFields(logrus.Fields{
					"direction":   conn.Stat().Direction,
					"multiAddr":   multiAddr,
					"activePeers": len(s.peers.Active()),
				})
				s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnecting)
				if err := reqFunc(context.Background(), conn.RemotePeer()); err != nil && err != io.EOF {
					log.WithError(err).Trace("Handshake failed")
					if err.Error() == "protocol not supported" {
						// This is only to ensure the smooth running of our testnets. This will not be
						// used in production.
						log.Debug("Not disconnecting peer with unsupported protocol. This may be the DHT node or relay.")
						s.host.ConnManager().Protect(conn.RemotePeer(), "relay/bootnode")
						s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
						return
					}
					s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnecting)
					if err := s.Disconnect(conn.RemotePeer()); err != nil {
						log.WithError(err).Error("Unable to disconnect from peer")
					}
					s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
					return
				}
				s.host.ConnManager().Protect(conn.RemotePeer(), "protocol")
				s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnected)
				log.Info("Peer connected")
			}()
		},
	})
}

// AddDisconnectionHandler disconnects from peers.  It handles updating the peer status.
// This also calls the handler responsible for maintaining other parts of the sync or p2p system.
func (s *Service) AddDisconnectionHandler(handler func(ctx context.Context, id peer.ID) error) {
	s.host.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			log := log.WithField("peer", conn.RemotePeer().Pretty())
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				priorState, err := s.peers.ConnectionState(conn.RemotePeer())
				if err != nil {
					// Can happen if the peer has already disconnected, so...
					priorState = peers.PeerDisconnected
				}
				s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnecting)
				ctx := context.Background()
				if err := handler(ctx, conn.RemotePeer()); err != nil {
					log.WithError(err).Error("Disconnect handler failed")
				}
				s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
				s.host.ConnManager().Unprotect(conn.RemotePeer(), "protocol")
				// Only log disconnections if we were fully connected.
				if priorState == peers.PeerConnected {
					log.WithField("active", len(s.peers.Active())).Info("Peer disconnected")
				}
			}()
		},
	})
}
