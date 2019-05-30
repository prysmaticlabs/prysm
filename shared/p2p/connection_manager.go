package p2p

import (
	"math"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	peer "github.com/libp2p/go-libp2p-peer"
)

// Reputation reward values.
const (
	RepRewardValidBlock       = 4
	RepRewardValidAttestation = 1

	RepPenalityInvalidProtobuf    = -1000
	RepPenalityInitialSyncFailure = -500
	RepPenalityInvalidBlock       = -10
	RepPenalityInvalidAttestation = -5
)

func optionConnectionManager(maxPeers int) libp2p.Option {
	if maxPeers < 5 {
		log.Warn("Max peers < 5. Defaulting to 5 max peers")
		maxPeers = 5
	}
	minPeers := int(math.Max(5, float64(maxPeers-5)))
	cm := connmgr.NewConnManager(minPeers, maxPeers, 20*time.Second)

	return libp2p.ConnectionManager(cm)
}

// Reputation adds (or subtracts) a given reward/penalty against a peer.
// Eventually, the lowest scoring peers will be pruned from the connections.
func (s *Server) Reputation(peer peer.ID, val int) {
	ti := s.host.ConnManager().GetTagInfo(peer)
	if ti != nil {
		val += ti.Value
	}
	s.host.ConnManager().TagPeer(peer, TagReputation, val)
}

// Disconnect will close all connections to the given peer.
func (s *Server) Disconnect(peer peer.ID) {
	if err := s.host.Network().ClosePeer(peer); err != nil {
		log.WithError(err).WithField("peer", peer.Pretty()).Error("Failed to close conn with peer")
	}
}
