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

const (
	// The time to wait for a status request.
	timeForStatus = 10 * time.Second
)

// AddConnectionHandler adds a callback function which handles the connection with a
// newly added peer. It performs a handshake with that peer by sending a hello request
// and validating the response from the peer.
func (s *Service) AddConnectionHandler(reqFunc func(ctx context.Context, id peer.ID) error,
	goodbyeFunc func(ctx context.Context, id peer.ID) error) {

	// Peer map and lock to keep track of current connection attempts.
	peerMap := make(map[peer.ID]bool)
	peerLock := new(sync.Mutex)

	// This is run at the start of each connection attempt, to ensure
	// that there aren't multiple inflight connection requests for the
	// same peer at once.
	peerHandshaking := func(id peer.ID) bool {
		peerLock.Lock()
		defer peerLock.Unlock()

		if peerMap[id] {
			repeatPeerConnections.Inc()
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
			log := log.WithField("peer", conn.RemotePeer().Pretty())
			remotePeer := conn.RemotePeer()
			disconnectFromPeer := func() {
				s.peers.SetConnectionState(remotePeer, peers.PeerDisconnecting)
				if err := s.Disconnect(remotePeer); err != nil {
					log.WithError(err).Error("Unable to disconnect from peer")
				}
				s.peers.SetConnectionState(remotePeer, peers.PeerDisconnected)
			}
			// Connection handler must be non-blocking as part of libp2p design.
			go func() {
				if peerHandshaking(remotePeer) {
					// Exit this if there is already another connection
					// request in flight.
					return
				}
				defer peerFinished(remotePeer)
				// Handle the various pre-existing conditions that will result in us not handshaking.
				peerConnectionState, err := s.peers.ConnectionState(remotePeer)
				if err == nil && (peerConnectionState == peers.PeerConnected || peerConnectionState == peers.PeerConnecting) {
					log.WithField("currentState", peerConnectionState).WithField("reason", "already active").Trace("Ignoring connection request")
					return
				}
				s.peers.Add(nil /* ENR */, remotePeer, conn.RemoteMultiaddr(), conn.Stat().Direction)
				if s.peers.IsBad(remotePeer) {
					log.WithField("reason", "bad peer").Trace("Ignoring connection request")
					disconnectFromPeer()
					return
				}
				validPeerConnection := func() {
					s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnected)
					// Go through the handshake process.
					multiAddr := fmt.Sprintf("%s/p2p/%s", conn.RemoteMultiaddr().String(), conn.RemotePeer().String())
					log.WithFields(logrus.Fields{
						"direction":   conn.Stat().Direction,
						"multiAddr":   multiAddr,
						"activePeers": len(s.peers.Active()),
					}).Info("Peer Connected")
				}

				// Do not perform handshake on inbound dials.
				if conn.Stat().Direction == network.DirInbound {
					_, err := s.peers.ChainState(remotePeer)
					peerExists := err == nil
					currentTime := roughtime.Now()

					// Wait for peer to initiate handshake
					time.Sleep(timeForStatus)

					// Exit if we are disconnected with the peer.
					if s.host.Network().Connectedness(remotePeer) != network.Connected {
						return
					}

					// If peer hasn't sent a status request, we disconnect with them
					if _, err := s.peers.ChainState(remotePeer); err == peers.ErrPeerUnknown {
						disconnectFromPeer()
						return
					}
					if peerExists {
						updated, err := s.peers.ChainStateLastUpdated(remotePeer)
						if err != nil {
							disconnectFromPeer()
							return
						}
						// exit if we don't receive any current status messages from
						// peer.
						if updated.IsZero() || !updated.After(currentTime) {
							disconnectFromPeer()
							return
						}
					}
					validPeerConnection()
					return
				}

				s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnecting)
				if err := reqFunc(context.Background(), conn.RemotePeer()); err != nil && err != io.EOF {
					log.WithError(err).Trace("Handshake failed")
					if err.Error() == "protocol not supported" {
						// This is only to ensure the smooth running of our testnets. This will not be
						// used in production.
						log.Debug("Not disconnecting peer with unsupported protocol. This may be the DHT node or relay.")
						s.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
						return
					}
					disconnectFromPeer()
					return
				}
				validPeerConnection()
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
				// Exit early if we are still connected to the peer.
				if net.Connectedness(conn.RemotePeer()) == network.Connected {
					return
				}
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
				// Only log disconnections if we were fully connected.
				if priorState == peers.PeerConnected {
					log.WithField("activePeers", len(s.peers.Active())).Info("Peer Disconnected")
				}
			}()
		},
	})
}
