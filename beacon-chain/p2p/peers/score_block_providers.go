package peers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
)

const (
	// DefaultBlockProviderProcessedBatchWeight is a default reward weight of a processed batch of blocks.
	DefaultBlockProviderProcessedBatchWeight = 0.05
	// DefaultBlockProviderDecayInterval defines how often the decaying routine is called.
	DefaultBlockProviderDecayInterval = 1 * time.Minute
)

// BlockProviderScorer represents block provider scoring service.
type BlockProviderScorer struct {
	ctx    context.Context
	config *BlockProviderScorerConfig
	store  *peerDataStore
	// maxProcessedBlocks defines maximum number of processed blocks attained by some peer during
	// the lifetime of a service. This is used to gauge relative performance of other peers.
	maxProcessedBlocks uint64
}

// BlockProviderScorerConfig holds configuration parameters for block providers scoring service.
type BlockProviderScorerConfig struct {
	// ProcessedBatchWeight defines a reward for a single processed batch of blocks.
	ProcessedBatchWeight float64
	// DecayInterval defines how often stats should be decayed.
	DecayInterval time.Duration
	// Decay specifies number of blocks subtracted from stats on each decay step.
	Decay uint64
}

// newBlockProviderScorer creates block provider scoring service.
func newBlockProviderScorer(
	ctx context.Context, store *peerDataStore, config *BlockProviderScorerConfig) *BlockProviderScorer {
	if config == nil {
		config = &BlockProviderScorerConfig{}
	}
	scorer := &BlockProviderScorer{
		ctx:                ctx,
		config:             config,
		store:              store,
		maxProcessedBlocks: uint64(flags.Get().BlockBatchLimit),
	}
	if scorer.config.ProcessedBatchWeight == 0.0 {
		scorer.config.ProcessedBatchWeight = DefaultBlockProviderProcessedBatchWeight
	}
	if scorer.config.DecayInterval == 0 {
		scorer.config.DecayInterval = DefaultBlockProviderDecayInterval
	}
	if scorer.config.Decay == 0 {
		scorer.config.Decay = uint64(flags.Get().BlockBatchLimit)
	}
	return scorer
}

// Score calculates and returns block provider score.
func (s *BlockProviderScorer) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.score(pid)
}

// score is a lock-free version of Score.
func (s *BlockProviderScorer) score(pid peer.ID) float64 {
	score := float64(0)
	peerData, ok := s.store.peers[pid]
	if !ok {
		return score
	}
	batchSize := uint64(flags.Get().BlockBatchLimit)
	if batchSize > 0 {
		processedBatches := float64(peerData.processedBlocks / batchSize)
		score += processedBatches * s.config.ProcessedBatchWeight
	}
	return math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor
}

// Params exposes scorer's parameters.
func (s *BlockProviderScorer) Params() *BlockProviderScorerConfig {
	return s.config
}

// IncrementProcessedBlocks increments the number of blocks that have been successfully processed.
func (s *BlockProviderScorer) IncrementProcessedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if cnt <= 0 {
		return
	}
	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].processedBlocks += cnt
	if s.store.peers[pid].processedBlocks > s.maxProcessedBlocks {
		s.maxProcessedBlocks = s.store.peers[pid].processedBlocks
	}
}

// ProcessedBlocks returns number of peer returned blocks that are successfully processed.
func (s *BlockProviderScorer) ProcessedBlocks(pid peer.ID) uint64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.processedBlocks(pid)
}

// processedBlocks is a lock-free version of ProcessedBlocks.
func (s *BlockProviderScorer) processedBlocks(pid peer.ID) uint64 {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.processedBlocks
	}
	return 0
}

// Decay updates block provider counters by decaying them.
// This urges peers to keep up the performance to continue getting a high score (and allows
// new peers to contest previously high scoring ones).
func (s *BlockProviderScorer) Decay() {
	s.store.Lock()
	defer s.store.Unlock()

	for _, peerData := range s.store.peers {
		if peerData.processedBlocks > s.config.Decay {
			peerData.processedBlocks -= s.config.Decay
		} else {
			peerData.processedBlocks = 0
		}
	}
}

// Sorted returns list of block providers sorted by score in descending order.
func (s *BlockProviderScorer) Sorted(pids []peer.ID) []peer.ID {
	s.store.Lock()
	defer s.store.Unlock()

	if len(pids) == 0 {
		return pids
	}
	scores := make(map[peer.ID]float64, len(pids))
	peers := make([]peer.ID, len(pids))
	for i, pid := range pids {
		scores[pid] = s.score(pid)
		peers[i] = pid
	}
	sort.Slice(peers, func(i, j int) bool {
		return scores[peers[i]] > scores[peers[j]]
	})
	return peers
}

// FormatScorePretty returns full scoring information in a human-readable format.
func (s *BlockProviderScorer) FormatScorePretty(pid peer.ID) string {
	s.store.RLock()
	defer s.store.RUnlock()
	score := s.score(pid)
	return fmt.Sprintf("[%0.1f%%, raw: %v,  blocks: %d/%d]",
		(score/s.MaxScore())*100, score, s.processedBlocks(pid), s.maxProcessedBlocks)
}

// MaxScore exposes maximum score attainable by peers.
func (s *BlockProviderScorer) MaxScore() float64 {
	score := s.Params().ProcessedBatchWeight
	batchSize := uint64(flags.Get().BlockBatchLimit)
	if batchSize > 0 {
		totalProcessedBatches := float64(s.maxProcessedBlocks / batchSize)
		score = totalProcessedBatches * s.config.ProcessedBatchWeight
	}
	return math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor
}
