package peers

import (
	"context"
	"math"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// PeerScorerManager keeps track of peer scorers that are used to calculate overall peer score.
type PeerScorerManager struct {
	ctx     context.Context
	store   *peerDataStore
	scorers struct {
		badResponsesScorer *BadResponsesScorer
	}
}

// PeerScorerConfig holds configuration parameters for scoring service.
type PeerScorerConfig struct {
	BadResponsesScorerConfig *BadResponsesScorerConfig
}

// newPeerScorerManager provides fully initialized peer scoring service.
func newPeerScorerManager(ctx context.Context, store *peerDataStore, config *PeerScorerConfig) *PeerScorerManager {
	mgr := &PeerScorerManager{
		ctx:   ctx,
		store: store,
	}

	mgr.scorers.badResponsesScorer = newBadResponsesScorer(ctx, store, config.BadResponsesScorerConfig)
	go mgr.loop(mgr.ctx)

	return mgr
}

// BadResponsesScorer exposes bad responses scoring service.
func (m *PeerScorerManager) BadResponsesScorer() *BadResponsesScorer {
	return m.scorers.badResponsesScorer
}

// Score returns calculated peer score across all tracked metrics.
func (m *PeerScorerManager) Score(pid peer.ID) float64 {
	m.store.RLock()
	defer m.store.RUnlock()

	score := float64(0)
	if _, ok := m.store.peers[pid]; !ok {
		return 0
	}
	score += m.scorers.badResponsesScorer.score(pid)
	return math.Round(score*10000) / 10000
}

// loop handles background tasks.
func (m *PeerScorerManager) loop(ctx context.Context) {
	decayBadResponsesStats := time.NewTicker(m.scorers.badResponsesScorer.Params().DecayInterval)
	defer decayBadResponsesStats.Stop()

	for {
		select {
		case <-decayBadResponsesStats.C:
			m.scorers.badResponsesScorer.Decay()
		case <-ctx.Done():
			return
		}
	}
}
