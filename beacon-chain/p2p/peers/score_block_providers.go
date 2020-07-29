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
	// DefaultBlockProviderReturnedBlocksWeight is a default weight of a returned/requested ratio in an overall score.
	DefaultBlockProviderReturnedBlocksWeight = 0.2
	// DefaultBlockProviderEmptyReturnedBatchPenalty is a default penalty for non-responsive peers.
	DefaultBlockProviderEmptyReturnedBatchPenalty = 0.0
	// DefaultBlockProviderProcessedBlocksWeight is a default weight of a processed/requested ratio in an overall score.
	DefaultBlockProviderProcessedBlocksWeight = 0.0
	// DefaultBlockProviderEmptyProcessedBatchPenalty is a default penalty for non-responsive peers.
	DefaultBlockProviderEmptyProcessedBatchPenalty = 0.0
	// DefaultBlockProviderDecayInterval defines how often block provider's stats should be decayed.
	DefaultBlockProviderDecayInterval = 1 * time.Minute
	// DefaultBlockProviderDecay specifies a decay factor (as a left-over percentage of the original value).
	DefaultBlockProviderDecay = 0.95
	// DefaultBlockProviderStartScore defines initial score before any stats updates takes place.
	// By setting this to positive value, peers are given a chance to be used for the first time.
	DefaultBlockProviderStartScore = 0.1
)

// BlockProviderScorer represents block provider scoring service.
type BlockProviderScorer struct {
	ctx    context.Context
	config *BlockProviderScorerConfig
	store  *peerDataStore
}

// BlockProviderScorerConfig holds configuration parameters for block providers scoring service.
type BlockProviderScorerConfig struct {
	// StartScore defines initial score from which peer starts. Set to positive to give peers an
	// opportunity to be selected for block fetching (allows new peers to start participating,
	// when there are already scored peers).
	StartScore float64
	// ReturnedBlocksWeight defines weight of a returned/requested ratio in overall an score.
	ReturnedBlocksWeight float64
	// EmptyReturnedBatchPenalty defines a penalty applied to score, if blocks were requested,
	// but none have been returned yet (to distinguish between non-responsive peers and peers that
	// haven't been requested any blocks yet). Penalty is applied per empty batch.
	EmptyReturnedBatchPenalty float64
	// ProcessedBlocksWeight defines weight of a processed/requested ratio in overall an score.
	ProcessedBlocksWeight float64
	// EmptyProcessedBatchPenalty defines a penalty applied to score, if blocks have been
	// requested, but none have been processed yet. Penalty is applied per empty batch.
	EmptyProcessedBatchPenalty float64
	// DecayInterval defines how often requested/returned/processed stats should be decayed.
	DecayInterval time.Duration
	// Decay specifies the factor (must be < 1.0) by which block provider's stats is decayed.
	Decay float64
}

// newBlockProviderScorer creates block provider scoring service.
func newBlockProviderScorer(
	ctx context.Context, store *peerDataStore, config *BlockProviderScorerConfig) *BlockProviderScorer {
	if config == nil {
		config = &BlockProviderScorerConfig{}
	}
	scorer := &BlockProviderScorer{
		ctx:    ctx,
		config: config,
		store:  store,
	}
	if scorer.config.StartScore == 0.0 {
		scorer.config.StartScore = DefaultBlockProviderStartScore
	}
	if scorer.config.ReturnedBlocksWeight == 0.0 {
		scorer.config.ReturnedBlocksWeight = DefaultBlockProviderReturnedBlocksWeight
	}
	if scorer.config.EmptyReturnedBatchPenalty == 0.0 {
		scorer.config.EmptyReturnedBatchPenalty = DefaultBlockProviderEmptyReturnedBatchPenalty
	}
	if scorer.config.ProcessedBlocksWeight == 0.0 {
		scorer.config.ProcessedBlocksWeight = DefaultBlockProviderProcessedBlocksWeight
	}
	if scorer.config.EmptyProcessedBatchPenalty == 0.0 {
		scorer.config.EmptyProcessedBatchPenalty = DefaultBlockProviderEmptyProcessedBatchPenalty
	}
	if scorer.config.DecayInterval == 0 {
		scorer.config.DecayInterval = DefaultBlockProviderDecayInterval
	}
	if scorer.config.Decay == 0.0 {
		scorer.config.Decay = DefaultBlockProviderDecay
	}
	return scorer
}

// Score calculates and returns total score based on returned and processed blocks.
func (s *BlockProviderScorer) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.score(pid)
}

// score is a lock-free version of Score.
func (s *BlockProviderScorer) score(pid peer.ID) float64 {
	score := s.Params().StartScore
	peerData, ok := s.store.peers[pid]
	if !ok {
		return score
	}
	if peerData.requestedBlocks > 0 {
		// Score returned/requested ratio. If there's more than 1 empty batch, apply as a penalty.
		returnedBlocksScore := float64(peerData.returnedBlocks) / float64(peerData.requestedBlocks)
		returnedBlocksScore = returnedBlocksScore * s.config.ReturnedBlocksWeight
		score += returnedBlocksScore

		emptyBatches := float64(peerData.requestedBlocks-peerData.returnedBlocks) / float64(flags.Get().BlockBatchLimit)
		if emptyBatches > 1 {
			score += s.config.EmptyReturnedBatchPenalty * emptyBatches
		}

		// Score processed/requested ratio. If there's more than 1 empty batch, apply as a penalty.
		processedBlocksScore := float64(peerData.processedBlocks) / float64(peerData.requestedBlocks)
		processedBlocksScore = processedBlocksScore * s.config.ProcessedBlocksWeight
		score += processedBlocksScore

		emptyBatches = float64(peerData.requestedBlocks-peerData.processedBlocks) / float64(flags.Get().BlockBatchLimit)
		if emptyBatches > 1 {
			score += s.config.EmptyProcessedBatchPenalty * emptyBatches
		}
	} else {
		// Boost peers that have never been selected.
		return s.MaxScore()
	}
	return math.Round(score*scoreRoundingFactor) / scoreRoundingFactor
}

// Params exposes scorer's parameters.
func (s *BlockProviderScorer) Params() *BlockProviderScorerConfig {
	return s.config
}

// IncrementRequestedBlocks increments the number of blocks that have been requested from peer.
func (s *BlockProviderScorer) IncrementRequestedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].requestedBlocks += cnt
}

// RequestedBlocks returns number of blocks requested from a peer.
func (s *BlockProviderScorer) RequestedBlocks(pid peer.ID) uint64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.requestedBlocks(pid)
}

// requestedBlocks is a lock-free version of RequestedBlocks.
func (s *BlockProviderScorer) requestedBlocks(pid peer.ID) uint64 {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.requestedBlocks
	}
	return 0
}

// IncrementReturnedBlocks increments the number of blocks that have been returned by peer.
func (s *BlockProviderScorer) IncrementReturnedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].returnedBlocks += cnt
}

// ReturnedBlocks returns number of blocks returned by a peer.
func (s *BlockProviderScorer) ReturnedBlocks(pid peer.ID) uint64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.returnedBlocks(pid)
}

// returnedBlocks is a lock-free version of ReturnedBlocks.
func (s *BlockProviderScorer) returnedBlocks(pid peer.ID) uint64 {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.returnedBlocks
	}
	return 0
}

// IncrementProcessedBlocks increments the number of blocks that have been successfully processed.
func (s *BlockProviderScorer) IncrementProcessedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].processedBlocks += cnt
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
// This urges peers to keep up the performance to get a high score (and allows new peers to contest previously high
// scoring ones).
func (s *BlockProviderScorer) Decay() {
	s.store.Lock()
	defer s.store.Unlock()

	for _, peerData := range s.store.peers {
		peerData.requestedBlocks = uint64(math.Round(float64(peerData.requestedBlocks) * s.config.Decay))
		// Once requested blocks stats drops to the half of batch size, reset stats.
		if peerData.requestedBlocks < uint64(flags.Get().BlockBatchLimit/2) {
			peerData.requestedBlocks = 0
			peerData.returnedBlocks = 0
			peerData.processedBlocks = 0
			continue
		}
		peerData.returnedBlocks = uint64(math.Round(float64(peerData.returnedBlocks) * s.config.Decay))
		peerData.processedBlocks = uint64(math.Round(float64(peerData.processedBlocks) * s.config.Decay))
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
	sort.SliceStable(peers, func(i, j int) bool {
		return scores[peers[i]] > scores[peers[j]]
	})
	return peers
}

// BlockProviderScorePretty returns full scoring information about a given peer.
func (s *BlockProviderScorer) BlockProviderScorePretty(pid peer.ID) string {
	s.store.Lock()
	defer s.store.Unlock()
	score := s.score(pid)
	return fmt.Sprintf("[%0.2f%%, raw: %v,  req: %d, ret: %d, proc: %d]",
		(score/s.MaxScore())*100, score,
		s.requestedBlocks(pid), s.returnedBlocks(pid), s.processedBlocks(pid))
}

// MaxScore exposes maximum score attainable by peers.
func (s *BlockProviderScorer) MaxScore() float64 {
	return s.Params().StartScore + s.config.ReturnedBlocksWeight + s.config.ProcessedBlocksWeight
}
