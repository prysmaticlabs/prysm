package peers

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

// BadResponsesThreshold returns the maximum number of bad responses a peer can provide before it is considered bad.
func (s *PeerScorer) BadResponsesThreshold() int {
	return s.params.BadResponsesThreshold
}

// BadResponses obtains the number of bad responses we have received from the given remote peer.
func (s *PeerScorer) BadResponses(pid peer.ID) (int, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if peerStats, ok := s.peerStats[pid]; ok {
		return peerStats.badResponses, nil
	}
	return -1, ErrPeerUnknown
}

// IncrementBadResponses increments the number of bad responses we have received from the given remote peer.
func (s *PeerScorer) IncrementBadResponses(pid peer.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	peerStats := s.fetch(pid)
	peerStats.badResponses++
}

// IsBadPeer states if the peer is to be considered bad.
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (s *PeerScorer) IsBadPeer(pid peer.ID) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if peerStats, ok := s.peerStats[pid]; ok {
		return peerStats.badResponses >= s.params.BadResponsesThreshold
	}
	return false
}

// IsGoodPeer states if the peer is to be considered good.
// If the peer is unknown this will return `true`.
func (s *PeerScorer) IsGoodPeer(pid peer.ID) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if peerStats, ok := s.peerStats[pid]; ok {
		return peerStats.badResponses < s.params.BadResponsesThreshold
	}
	return true
}

// BadPeers returns the peers that are bad.
func (s *PeerScorer) BadPeers() []peer.ID {
	s.lock.RLock()
	defer s.lock.RUnlock()

	badPeers := make([]peer.ID, 0)
	for pid, peerStats := range s.peerStats {
		if peerStats.badResponses >= s.params.BadResponsesThreshold {
			badPeers = append(badPeers, pid)
		}
	}
	return badPeers
}

// decayBadResponsesStats reduces the bad responses of all peers, giving reformed peers a chance to join the network.
// This can be run periodically, although note that each time it runs it does give all bad peers another chance as well
// to clog up the network with bad responses, so should not be run too frequently; once an hour would be reasonable.
func (s *PeerScorer) decayBadResponsesStats() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, peerStats := range s.peerStats {
		if peerStats.badResponses > 0 {
			peerStats.badResponses--
		}
	}
}
