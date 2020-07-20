package peers

import (
	"context"
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

// PeerScorer keeps track of peer counters that are used to calculate peer score.
type PeerScorer struct {
	ctx    context.Context
	config *PeerScorerConfig
	store  *peerDataStore
}

// PeerScorerConfig holds configuration parameters for scoring service.
type PeerScorerConfig struct {
	// BadResponsesThreshold specifies number of bad responses tolerated, before peer is banned.
	BadResponsesThreshold     int
	BadResponsesWeight        float64
	BadResponsesDecayInterval time.Duration
}

// newPeerScorer provides fully initialized peer scoring service.
func newPeerScorer(ctx context.Context, store *peerDataStore, config *PeerScorerConfig) *PeerScorer {
	scorer := &PeerScorer{
		ctx:    ctx,
		config: config,
		store:  store,
	}
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

	var score float64
	if _, ok := s.store.peers[pid]; !ok {
		return 0
	}

	badResponsesScore := float64(s.store.peers[pid].badResponsesCount) / float64(s.config.BadResponsesThreshold)
	badResponsesScore = badResponsesScore * s.config.BadResponsesWeight
	score += badResponsesScore

	return score
}

// Params exposes peer scorer parameters.
func (s *PeerScorer) Params() *PeerScorerConfig {
	return s.config
}

// loop handles background tasks.
func (s *PeerScorer) loop(ctx context.Context) {
	decayBadResponsesScores := time.NewTicker(s.config.BadResponsesDecayInterval)
	defer decayBadResponsesScores.Stop()

	for {
		select {
		case <-decayBadResponsesScores.C:
			s.DecayBadResponsesStats()
		case <-ctx.Done():
			return
		}
	}
}
