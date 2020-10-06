package scorers

import (
	"context"
	"math"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/data"
)

// ScoreRoundingFactor defines how many digits to keep in decimal part.
// This parameter is used in math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor.
const ScoreRoundingFactor = 10000

// Service manages peer scorers that are used to calculate overall peer score.
type Service struct {
	ctx     context.Context
	store   *data.Store
	scorers struct {
		badResponsesScorer  *BadResponsesScorer
		blockProviderScorer *BlockProviderScorer
	}
}

// Config holds configuration parameters for scoring service.
type Config struct {
	BadResponsesScorerConfig  *BadResponsesScorerConfig
	BlockProviderScorerConfig *BlockProviderScorerConfig
}

// NewService provides fully initialized peer scoring service.
func NewService(ctx context.Context, store *data.Store, config *Config) *Service {
	mgr := &Service{
		ctx:   ctx,
		store: store,
	}
	mgr.scorers.badResponsesScorer = newBadResponsesScorer(ctx, store, config.BadResponsesScorerConfig)
	mgr.scorers.blockProviderScorer = newBlockProviderScorer(ctx, store, config.BlockProviderScorerConfig)
	go mgr.loop(mgr.ctx)

	return mgr
}

// BadResponsesScorer exposes bad responses scoring service.
func (m *Service) BadResponsesScorer() *BadResponsesScorer {
	return m.scorers.badResponsesScorer
}

// BlockProviderScorer exposes block provider scoring service.
func (m *Service) BlockProviderScorer() *BlockProviderScorer {
	return m.scorers.blockProviderScorer
}

// Score returns calculated peer score across all tracked metrics.
func (m *Service) Score(pid peer.ID) float64 {
	m.store.RLock()
	defer m.store.RUnlock()

	score := float64(0)
	if _, ok := m.store.PeerData(pid); !ok {
		return 0
	}
	score += m.scorers.badResponsesScorer.score(pid)
	score += m.scorers.blockProviderScorer.score(pid)
	return math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor
}

// loop handles background tasks.
func (m *Service) loop(ctx context.Context) {
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
