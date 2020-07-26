package peers

import (
	"context"
	"math"
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
	// DefaultBlockProviderReturnedBlocksWeight is a default weight of a returned/requested ratio in an overall score.
	DefaultBlockProviderReturnedBlocksWeight = 0.2
	// DefaultBlockProviderNoReturnedBlocksPenalty is a default penalty for non-responsive peers.
	DefaultBlockProviderNoReturnedBlocksPenalty = -0.02
	// DefaultBlockProviderProcessedBlocksWeight is a default weight of a processed/requested ratio in an overall score.
	DefaultBlockProviderProcessedBlocksWeight = 0.2
	// DefaultBlockProviderNoProcessedBlocksPenalty is a default penalty for non-responsive peers.
	DefaultBlockProviderNoProcessedBlocksPenalty = -0.02
	// DefaultBlockProviderDecayInterval defines how often block provider's stats should be decayed.
	DefaultBlockProviderDecayInterval = 5 * time.Minute
	// DefaultBlockProviderDecay specifies a decay factor (as a left-over percentage of the original value).
	DefaultBlockProviderDecay = 0.95
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
	// BlockProviderNoReturnedBlocksPenalty defines a penalty applied to score, if blocks were requested,
	// but none have been returned yet (to distinguish between non-responsive peers and peers that
	// haven't been requested any blocks yet).
	BlockProviderNoReturnedBlocksPenalty float64
	// BlockProviderProcessedBlocksWeight defines weight of a processed/requested ratio in overall an score.
	BlockProviderProcessedBlocksWeight float64
	// BlockProviderNoProcessedBlocksPenalty defines a penalty applied to score, if blocks have been
	// requested, but none have been processed yet.
	BlockProviderNoProcessedBlocksPenalty float64
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
	if scorer.config.BlockProviderNoReturnedBlocksPenalty == 0.0 {
		scorer.config.BlockProviderNoReturnedBlocksPenalty = DefaultBlockProviderNoReturnedBlocksPenalty
	}
	if scorer.config.BlockProviderProcessedBlocksWeight == 0.0 {
		scorer.config.BlockProviderProcessedBlocksWeight = DefaultBlockProviderProcessedBlocksWeight
	}
	if scorer.config.BlockProviderNoProcessedBlocksPenalty == 0.0 {
		scorer.config.BlockProviderNoProcessedBlocksPenalty = DefaultBlockProviderNoProcessedBlocksPenalty
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
