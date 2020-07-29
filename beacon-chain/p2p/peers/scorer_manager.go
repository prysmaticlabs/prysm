package peers

import (
	"context"
	"math"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// scoreRoundingFactor defines how many digits to keep in decimal part.
// This parameter is used in math.Round(score*scoreRoundingFactor) / scoreRoundingFactor.
const scoreRoundingFactor = 10000

// PeerScorerManager keeps track of peer scorers that are used to calculate overall peer score.
type PeerScorerManager struct {
	ctx     context.Context
	store   *peerDataStore
	scorers struct {
		badResponsesScorer  *BadResponsesScorer
		blockProviderScorer *BlockProviderScorer
	}
}

// PeerScorerConfig holds configuration parameters for scoring service.
type PeerScorerConfig struct {
	BadResponsesScorerConfig  *BadResponsesScorerConfig
	BlockProviderScorerConfig *BlockProviderScorerConfig
}

// newPeerScorerManager provides fully initialized peer scoring service.
func newPeerScorerManager(ctx context.Context, store *peerDataStore, config *PeerScorerConfig) *PeerScorerManager {
	mgr := &PeerScorerManager{
		ctx:   ctx,
		store: store,
	}
	mgr.scorers.badResponsesScorer = newBadResponsesScorer(ctx, store, config.BadResponsesScorerConfig)
	mgr.scorers.blockProviderScorer = newBlockProviderScorer(ctx, store, config.BlockProviderScorerConfig)
	go mgr.loop(mgr.ctx)

	return mgr
}

// BadResponsesScorer exposes bad responses scoring service.
func (m *PeerScorerManager) BadResponsesScorer() *BadResponsesScorer {
	return m.scorers.badResponsesScorer
}

// BlockProviderScorer exposes block provider scoring service.
func (m *PeerScorerManager) BlockProviderScorer() *BlockProviderScorer {
	return m.scorers.blockProviderScorer
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
	score += m.scorers.blockProviderScorer.score(pid)
	return math.Round(score*scoreRoundingFactor) / scoreRoundingFactor
}

// loop handles background tasks.
func (m *PeerScorerManager) loop(ctx context.Context) {
	decayBadResponsesStats := time.NewTicker(m.scorers.badResponsesScorer.Params().DecayInterval)
	defer decayBadResponsesStats.Stop()
	decayBlockProviderStats := time.NewTicker(m.scorers.blockProviderScorer.Params().DecayInterval)
	defer decayBlockProviderStats.Stop()

	for {
		select {
		case <-decayBadResponsesStats.C:
			m.scorers.badResponsesScorer.Decay()
		case <-decayBlockProviderStats.C:
			m.scorers.blockProviderScorer.Decay()
		case <-ctx.Done():
			return
		}
	}
}
