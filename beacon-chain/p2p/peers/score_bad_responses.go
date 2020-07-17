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
	s.store.RLock()
	defer s.store.RUnlock()
	return s.badResponses(pid)
}

// badResponses is a lock-free version of BadResponses.
func (s *PeerScorer) badResponses(pid peer.ID) (int, error) {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.badResponsesCount, nil
	}
	return -1, ErrPeerUnknown
}

// IncrementBadResponses increments the number of bad responses we have received from the given remote peer.
// If peer doesn't exist this method is no-op.
func (s *PeerScorer) IncrementBadResponses(pid peer.ID) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		return
	}
	s.store.peers[pid].badResponsesCount++
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
		return peerData.badResponsesCount >= s.params.BadResponsesThreshold
	}
	return false
}

// BadPeers returns the peers that are bad.
func (s *PeerScorer) BadPeers() []peer.ID {
	s.store.RLock()
	defer s.store.RUnlock()

	badPeers := make([]peer.ID, 0)
	for pid, peerData := range s.store.peers {
		if peerData.badResponsesCount >= s.params.BadResponsesThreshold {
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
		if peerData.badResponsesCount > 0 {
			peerData.badResponsesCount--
		}
	}
}
