package scorers

import (
	"context"
	"math"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/peerdata"
)

// ScoreRoundingFactor defines how many digits to keep in decimal part.
// This parameter is used in math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor.
const ScoreRoundingFactor = 10000

// Service manages peer scorers that are used to calculate overall peer score.
type Service struct {
	ctx     context.Context
	store   *peerdata.Store
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
func NewService(ctx context.Context, store *peerdata.Store, config *Config) *Service {
	s := &Service{
		ctx:   ctx,
		store: store,
	}
	s.scorers.badResponsesScorer = newBadResponsesScorer(ctx, store, config.BadResponsesScorerConfig)
	s.scorers.blockProviderScorer = newBlockProviderScorer(ctx, store, config.BlockProviderScorerConfig)
	go s.loop(s.ctx)

	return s
}

// BadResponsesScorer exposes bad responses scoring service.
func (s *Service) BadResponsesScorer() *BadResponsesScorer {
	return s.scorers.badResponsesScorer
}

// BlockProviderScorer exposes block provider scoring service.
func (s *Service) BlockProviderScorer() *BlockProviderScorer {
	return s.scorers.blockProviderScorer
}

// Score returns calculated peer score across all tracked metrics.
func (s *Service) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()

	score := float64(0)
	if _, ok := s.store.PeerData(pid); !ok {
		return 0
	}
	score += s.scorers.badResponsesScorer.score(pid)
	score += s.scorers.blockProviderScorer.score(pid)
	return math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor
}

// loop handles background tasks.
func (s *Service) loop(ctx context.Context) {
	decayBadResponsesStats := time.NewTicker(s.scorers.badResponsesScorer.Params().DecayInterval)
	defer decayBadResponsesStats.Stop()
	decayBlockProviderStats := time.NewTicker(s.scorers.blockProviderScorer.Params().DecayInterval)
	defer decayBlockProviderStats.Stop()

	for {
		select {
		case <-decayBadResponsesStats.C:
			s.scorers.badResponsesScorer.Decay()
		case <-decayBlockProviderStats.C:
			s.scorers.blockProviderScorer.Decay()
		case <-ctx.Done():
			return
		}
	}
}
