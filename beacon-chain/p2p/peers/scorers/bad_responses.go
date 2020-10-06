package scorers

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/peerdata"
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

// BadResponsesScorer represents bad responses scoring service.
type BadResponsesScorer struct {
	ctx    context.Context
	config *BadResponsesScorerConfig
	store  *peerdata.Store
}

// BadResponsesScorerConfig holds configuration parameters for bad response scoring service.
type BadResponsesScorerConfig struct {
	// Threshold specifies number of bad responses tolerated, before peer is banned.
	Threshold int
	// Weight defines weight of bad response/threshold ratio on overall score.
	Weight float64
	// DecayInterval specifies how often bad response stats should be decayed.
	DecayInterval time.Duration
}

// newBadResponsesScorer creates new bad responses scoring service.
func newBadResponsesScorer(
	ctx context.Context, store *peerdata.Store, config *BadResponsesScorerConfig) *BadResponsesScorer {
	if config == nil {
		config = &BadResponsesScorerConfig{}
	}
	scorer := &BadResponsesScorer{
		ctx:    ctx,
		config: config,
		store:  store,
	}
	if scorer.config.Threshold == 0 {
		scorer.config.Threshold = DefaultBadResponsesThreshold
	}
	if scorer.config.Weight == 0.0 {
		scorer.config.Weight = DefaultBadResponsesWeight
	}
	if scorer.config.DecayInterval == 0 {
		scorer.config.DecayInterval = DefaultBadResponsesDecayInterval
	}
	return scorer
}

// Score returns score (penalty) of bad responses peer produced.
func (s *BadResponsesScorer) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.score(pid)
}

// score is a lock-free version of ScoreBadResponses.
func (s *BadResponsesScorer) score(pid peer.ID) float64 {
	score := float64(0)
	peerData, ok := s.store.PeerData(pid)
	if !ok {
		return score
	}
	if peerData.BadResponses > 0 {
		score = float64(peerData.BadResponses) / float64(s.config.Threshold)
		score = score * s.config.Weight
	}
	return score
}

// Params exposes scorer's parameters.
func (s *BadResponsesScorer) Params() *BadResponsesScorerConfig {
	return s.config
}

// Count obtains the number of bad responses we have received from the given remote peer.
func (s *BadResponsesScorer) Count(pid peer.ID) (int, error) {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.count(pid)
}

// count is a lock-free version of Count.
func (s *BadResponsesScorer) count(pid peer.ID) (int, error) {
	if peerData, ok := s.store.PeerData(pid); ok {
		return peerData.BadResponses, nil
	}
	return -1, peerdata.ErrPeerUnknown
}

// Increment increments the number of bad responses we have received from the given remote peer.
// If peer doesn't exist this method is no-op.
func (s *BadResponsesScorer) Increment(pid peer.ID) {
	s.store.Lock()
	defer s.store.Unlock()

	peerData, ok := s.store.PeerData(pid)
	if !ok {
		s.store.SetPeerData(pid, &peerdata.PeerData{
			BadResponses: 1,
		})
		return
	}
	peerData.BadResponses++
}

// IsBadPeer states if the peer is to be considered bad.
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (s *BadResponsesScorer) IsBadPeer(pid peer.ID) bool {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.isBadPeer(pid)
}

// isBadPeer is lock-free version of IsBadPeer.
func (s *BadResponsesScorer) isBadPeer(pid peer.ID) bool {
	if peerData, ok := s.store.PeerData(pid); ok {
		return peerData.BadResponses >= s.config.Threshold
	}
	return false
}

// BadPeers returns the peers that are bad.
func (s *BadResponsesScorer) BadPeers() []peer.ID {
	s.store.RLock()
	defer s.store.RUnlock()

	badPeers := make([]peer.ID, 0)
	for pid := range s.store.Peers() {
		if s.isBadPeer(pid) {
			badPeers = append(badPeers, pid)
		}
	}
	return badPeers
}

// Decay reduces the bad responses of all peers, giving reformed peers a chance to join the network.
// This can be run periodically, although note that each time it runs it does give all bad peers another chance as well
// to clog up the network with bad responses, so should not be run too frequently; once an hour would be reasonable.
func (s *BadResponsesScorer) Decay() {
	s.store.Lock()
	defer s.store.Unlock()

	for _, peerData := range s.store.Peers() {
		if peerData.BadResponses > 0 {
			peerData.BadResponses--
		}
	}
}
