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
	params *PeerScorerParams
	store  *peerDataStore
}

// PeerScorerParams holds configuration parameters for scoring service.
type PeerScorerParams struct {
	BadResponsesThreshold     int
	BadResponsesWeight        float64
	BadResponsesDecayInterval time.Duration
}

// newPeerScorer provides fully initialized peer scoring service.
func newPeerScorer(ctx context.Context, store *peerDataStore, params *PeerScorerParams) *PeerScorer {
	scorer := &PeerScorer{
		ctx:    ctx,
		params: params,
		store:  store,
	}
	if scorer.params.BadResponsesThreshold <= 0 {
		scorer.params.BadResponsesThreshold = DefaultBadResponsesThreshold
	}
	if scorer.params.BadResponsesWeight == 0.0 {
		scorer.params.BadResponsesWeight = DefaultBadResponsesWeight
	}
	if scorer.params.BadResponsesDecayInterval <= 0 {
		scorer.params.BadResponsesDecayInterval = DefaultBadResponsesDecayInterval
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

	badResponsesScore := float64(s.store.peers[pid].badResponsesCount) / float64(s.params.BadResponsesThreshold)
	badResponsesScore = badResponsesScore * s.params.BadResponsesWeight
	score += badResponsesScore

	return score
}

// Params exposes peer scorer parameters.
func (s *PeerScorer) Params() *PeerScorerParams {
	return s.params
}

// loop handles background tasks.
func (s *PeerScorer) loop(ctx context.Context) {
	decayBadResponsesScores := time.NewTicker(s.params.BadResponsesDecayInterval)
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
