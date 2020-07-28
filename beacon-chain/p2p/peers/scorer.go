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

	// BlockProviderReturnedBlocksWeight defines weight of a returned/requested ratio in overall an score.
	BlockProviderReturnedBlocksWeight float64
	// BlockProviderEmptyReturnedBatchPenalty defines a penalty applied to score, if blocks were requested,
	// but none have been returned yet (to distinguish between non-responsive peers and peers that
	// haven't been requested any blocks yet). Penalty is applied per empty batch.
	BlockProviderEmptyReturnedBatchPenalty float64
	// BlockProviderProcessedBlocksWeight defines weight of a processed/requested ratio in overall an score.
	BlockProviderProcessedBlocksWeight float64
	// BlockProviderEmptyProcessedBatchPenalty defines a penalty applied to score, if blocks have been
	// requested, but none have been processed yet. Penalty is applied per empty batch.
	BlockProviderEmptyProcessedBatchPenalty float64
	// BlockProviderDecayInterval defines how often requested/returned/processed stats should be decayed.
	BlockProviderDecayInterval time.Duration
	// BlockProviderDecay specifies the factor (must be < 1.0) by which block provider's stats is decayed.
	BlockProviderDecay float64
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

	// Peer providers stats parameters.
	if scorer.config.BlockProviderReturnedBlocksWeight == 0.0 {
		scorer.config.BlockProviderReturnedBlocksWeight = DefaultBlockProviderReturnedBlocksWeight
	}
	if scorer.config.BlockProviderEmptyReturnedBatchPenalty == 0.0 {
		scorer.config.BlockProviderEmptyReturnedBatchPenalty = DefaultBlockProviderEmptyReturnedBatchPenalty
	}
	if scorer.config.BlockProviderProcessedBlocksWeight == 0.0 {
		scorer.config.BlockProviderProcessedBlocksWeight = DefaultBlockProviderProcessedBlocksWeight
	}
	if scorer.config.BlockProviderEmptyProcessedBatchPenalty == 0.0 {
		scorer.config.BlockProviderEmptyProcessedBatchPenalty = DefaultBlockProviderEmptyProcessedBatchPenalty
	}
	if scorer.config.BlockProviderDecayInterval == 0 {
		scorer.config.BlockProviderDecayInterval = DefaultBlockProviderDecayInterval
	}
	if scorer.config.BlockProviderDecay == 0.0 {
		scorer.config.BlockProviderDecay = DefaultBlockProviderDecay
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
	score += s.scoreBlockProvider(pid)
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
	decayBlockProvidersStats := time.NewTicker(s.config.BlockProviderDecayInterval)
	defer decayBlockProvidersStats.Stop()

	for {
		select {
		case <-decayBadResponsesStats.C:
			s.DecayBadResponsesStats()
		case <-decayBlockProvidersStats.C:
			s.DecayBlockProvidersStats()
		case <-ctx.Done():
			return
		}
	}
}
