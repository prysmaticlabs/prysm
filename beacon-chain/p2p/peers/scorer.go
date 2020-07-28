package peers

import (
	"context"
	"math"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// PeerScorer keeps track of peer counters that are used to calculate peer score.
type PeerScorer struct {
	ctx    context.Context
	config *PeerScorerConfig
	store  *peerDataStore
}

// PeerScorerConfig holds configuration parameters for scoring service.
type PeerScorerConfig struct {
	// BadResponsesThreshold specifies number of bad responses tolerated, before peer is banned.
	BadResponsesThreshold int
	// BadResponsesWeight defines weight of bad response/threshold ratio on overall score.
	BadResponsesWeight float64
	// BadResponsesDecayInterval specifies how often bad response stats should be decayed.
	BadResponsesDecayInterval time.Duration
}

// newPeerScorer provides fully initialized peer scoring service.
func newPeerScorer(ctx context.Context, store *peerDataStore, config *PeerScorerConfig) *PeerScorer {
	scorer := &PeerScorer{
		ctx:    ctx,
		config: config,
		store:  store,
	}

	// Bad responses stats parameters.
	if scorer.config.BadResponsesThreshold == 0 {
		scorer.config.BadResponsesThreshold = DefaultBadResponsesThreshold
	}
	if scorer.config.BadResponsesWeight == 0.0 {
		scorer.config.BadResponsesWeight = DefaultBadResponsesWeight
	}
	if scorer.config.BadResponsesDecayInterval == 0 {
		scorer.config.BadResponsesDecayInterval = DefaultBadResponsesDecayInterval
	}

	go scorer.loop(scorer.ctx)

	return scorer
}

// Score returns calculated peer score across all tracked metrics.
func (s *PeerScorer) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()

	score := float64(0)
	if _, ok := s.store.peers[pid]; !ok {
		return 0
	}
	score += s.scoreBadResponses(pid)
	return math.Round(score*10000) / 10000
}

// Params exposes peer scorer parameters.
func (s *PeerScorer) Params() *PeerScorerConfig {
	return s.config
}

// loop handles background tasks.
func (s *PeerScorer) loop(ctx context.Context) {
	decayBadResponsesStats := time.NewTicker(s.config.BadResponsesDecayInterval)
	defer decayBadResponsesStats.Stop()

	for {
		select {
		case <-decayBadResponsesStats.C:
			s.DecayBadResponsesStats()
		case <-ctx.Done():
			return
		}
	}
}
