package p2p

import (
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-peer"
)

const (
	RepRewardValidBlock       = 4
	RepRewardValidAttestation = 1

	RepPenalityInvalidProtobuf    = -1000
	RepPenalityInitialSyncFailure = -500
	RepPenalityInvalidBlock       = -10
	RepPenalityInvalidAttestation = -5
)

func optionConnectionManager() libp2p.Option {
	cm := connmgr.NewConnManager(25, 30, 5*time.Minute)

	return libp2p.ConnectionManager(cm)
}

func (s *Server) Reputation(peer peer.ID, val int) {
	s.host.ConnManager().TagPeer(peer, TagReputation, val)
}

func (s *Server) Disconnect(peer peer.ID) {
	if err := s.host.Network().ClosePeer(peer); err != nil {
		log.WithError(err).WithField("peer", peer.Pretty()).Error("Failed to close conn with peer")
	}
}
