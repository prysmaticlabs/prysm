package peers

import (
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	// DefaultBadResponsesThreshold defines how many bad responses to tolerate before peer is deemed bad.
	DefaultBadResponsesThreshold = 6
	// DefaultBadResponsesWeight is a default weight. Since score represents penalty, it has negative weight.
	DefaultBadResponsesWeight = -1.0
	// DefaultBadResponsesDecayInterval defines how often to decay previous statistics.
	// Every interval bad responses counter will be decremented by 1.
	DefaultBadResponsesDecayInterval = time.Hour
)

// ScoreBadResponses returns score (penalty) of bad responses peer produced.
func (s *PeerScorer) ScoreBadResponses(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.scoreBadResponses(pid)
}

// scoreBadResponses is a lock-free version of ScoreBadResponses.
func (s *PeerScorer) scoreBadResponses(pid peer.ID) float64 {
	score := float64(0)
	peerData, ok := s.store.peers[pid]
	if !ok {
		return score
	}
	if peerData.badResponses > 0 {
		score = float64(peerData.badResponses) / float64(s.config.BadResponsesThreshold)
		score = score * s.config.BadResponsesWeight
	}
	return score
}

// BadResponsesThreshold returns the maximum number of bad responses a peer can provide before it is considered bad.
func (s *PeerScorer) BadResponsesThreshold() int {
	return s.config.BadResponsesThreshold
}

// BadResponses obtains the number of bad responses we have received from the given remote peer.
func (s *PeerScorer) BadResponses(pid peer.ID) (int, error) {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.badResponses(pid)
}

// badResponses is a lock-free version of BadResponses.
func (s *PeerScorer) badResponses(pid peer.ID) (int, error) {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.badResponses, nil
	}
	return -1, ErrPeerUnknown
}

// IncrementBadResponses increments the number of bad responses we have received from the given remote peer.
// If peer doesn't exist this method is no-op.
func (s *PeerScorer) IncrementBadResponses(pid peer.ID) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{
			badResponses: 1,
		}
		return
	}
	s.store.peers[pid].badResponses++
}

// IsBadPeer states if the peer is to be considered bad.
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (s *PeerScorer) IsBadPeer(pid peer.ID) bool {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.isBadPeer(pid)
}

// isBadPeer is lock-free version of IsBadPeer.
func (s *PeerScorer) isBadPeer(pid peer.ID) bool {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.badResponses >= s.config.BadResponsesThreshold
	}
	return false
}

// BadPeers returns the peers that are bad.
func (s *PeerScorer) BadPeers() []peer.ID {
	s.store.RLock()
	defer s.store.RUnlock()

	badPeers := make([]peer.ID, 0)
	for pid := range s.store.peers {
		if s.isBadPeer(pid) {
			badPeers = append(badPeers, pid)
		}
	}
	return badPeers
}

// DecayBadResponsesStats reduces the bad responses of all peers, giving reformed peers a chance to join the network.
// This can be run periodically, although note that each time it runs it does give all bad peers another chance as well
// to clog up the network with bad responses, so should not be run too frequently; once an hour would be reasonable.
func (s *PeerScorer) DecayBadResponsesStats() {
	s.store.Lock()
	defer s.store.Unlock()

	for _, peerData := range s.store.peers {
		if peerData.badResponses > 0 {
			peerData.badResponses--
		}
	}
}
