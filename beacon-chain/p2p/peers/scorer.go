package peers

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	defaultBadResponsesThreshold     = 6
	defaultBadResponsesWeight        = -0.75
	defaultBadResponsesDecayInterval = time.Hour
)

// PeerScorer keeps track of peer counters that are used to calculate peer score.
type PeerScorer struct {
	lock      sync.RWMutex
	ctx       context.Context
	params    *PeerScorerParams
	peerStats map[peer.ID]*peerScorerStats
}

// peerScorerStats holds peer counters and statistics that is used in per per score calculation.
type peerScorerStats struct {
	badResponses int
}

// PeerScorerParams holds configuration parameters for scoring service.
type PeerScorerParams struct {
	BadResponsesThreshold     int
	BadResponsesWeight        float64
	BadResponsesDecayInterval time.Duration
}

// NewPeerScorer provides fully initialized peer scoring service.
func NewPeerScorer(ctx context.Context, params *PeerScorerParams) *PeerScorer {
	scorer := &PeerScorer{
		ctx:       ctx,
		params:    params,
		peerStats: make(map[peer.ID]*peerScorerStats),
	}
	if scorer.params.BadResponsesThreshold <= 0 {
		scorer.params.BadResponsesThreshold = defaultBadResponsesThreshold
	}
	if scorer.params.BadResponsesWeight == 0.0 {
		scorer.params.BadResponsesWeight = defaultBadResponsesWeight
	}
	if scorer.params.BadResponsesDecayInterval <= 0 {
		scorer.params.BadResponsesDecayInterval = defaultBadResponsesDecayInterval
	}

	go scorer.loop(scorer.ctx)

	return scorer
}

// AddPeer adds peer record to peer stats map.
func (s *PeerScorer) AddPeer(pid peer.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Fetch creates peer stats object (if it doesn't already exist).
	s.fetch(pid)
}

// Score returns calculated peer score across all tracked metrics.
func (s *PeerScorer) Score(pid peer.ID) float64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var score float64
	peerStats := s.fetch(pid)

	badResponsesScore := float64(peerStats.badResponses) / float64(s.params.BadResponsesThreshold)
	badResponsesScore = badResponsesScore * s.params.BadResponsesWeight
	score += badResponsesScore

	return score
}

// loop handles background tasks.
func (s *PeerScorer) loop(ctx context.Context) {
	decayBadResponsesScores := time.NewTicker(s.params.BadResponsesDecayInterval)
	defer decayBadResponsesScores.Stop()

	for {
		select {
		case <-decayBadResponsesScores.C:
			s.decayBadResponsesStats()
		case <-ctx.Done():
			return
		}
	}
}

// fetch is a helper function that fetches a peer stats, possibly creating it.
// This method must be called after s.lock is locked.
func (s *PeerScorer) fetch(pid peer.ID) *peerScorerStats {
	if _, ok := s.peerStats[pid]; !ok {
		s.peerStats[pid] = &peerScorerStats{}
	}
	return s.peerStats[pid]
}
