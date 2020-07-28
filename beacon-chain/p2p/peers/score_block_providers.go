package peers

import (
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
	DefaultBlockProviderEmptyReturnedBatchPenalty = -0.02
	// DefaultBlockProviderProcessedBlocksWeight is a default weight of a processed/requested ratio in an overall score.
	DefaultBlockProviderProcessedBlocksWeight = 0.0
	// DefaultBlockProviderEmptyProcessedBatchPenalty is a default penalty for non-responsive peers.
	DefaultBlockProviderEmptyProcessedBatchPenalty = 0.0
	// DefaultBlockProviderDecayInterval defines how often block provider's stats should be decayed.
	DefaultBlockProviderDecayInterval = 1 * time.Minute
	// DefaultBlockProviderDecay specifies a decay factor (as a left-over percentage of the original value).
	DefaultBlockProviderDecay = 0.95
	// blockProviderStartScore defines initial score before any stats updates takes place.
	// By setting this to positive value, peers are given a chance to be used for the first time.
	blockProviderStartScore = 0.1
)

// ScoreBlockProvider calculates and returns total score based on returned and processed blocks.
func (s *PeerScorer) ScoreBlockProvider(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.scoreBlockProvider(pid)
}

// scoreBlockProvider is a lock-free version of ScoreBlockProvider.
func (s *PeerScorer) scoreBlockProvider(pid peer.ID) float64 {
	score := s.BlockProviderStartScore()
	peerData, ok := s.store.peers[pid]
	if !ok {
		return score
	}
	if peerData.requestedBlocks > 0 {
		// Score returned/requested ratio. If there's more than 1 empty batch, apply as a penalty.
		returnedBlocksScore := float64(peerData.returnedBlocks) / float64(peerData.requestedBlocks)
		returnedBlocksScore = returnedBlocksScore * s.config.BlockProviderReturnedBlocksWeight
		score += returnedBlocksScore

		emptyBatches := float64(peerData.requestedBlocks-peerData.returnedBlocks) / float64(flags.Get().BlockBatchLimit)
		if emptyBatches > 1 {
			score += s.config.BlockProviderEmptyReturnedBatchPenalty * emptyBatches
		}

		// Score processed/requested ratio. If there's more than 1 empty batch, apply as a penalty.
		processedBlocksScore := float64(peerData.processedBlocks) / float64(peerData.requestedBlocks)
		processedBlocksScore = processedBlocksScore * s.config.BlockProviderProcessedBlocksWeight
		score += processedBlocksScore

		emptyBatches = float64(peerData.requestedBlocks-peerData.processedBlocks) / float64(flags.Get().BlockBatchLimit)
		if emptyBatches > 1 {
			score += s.config.BlockProviderEmptyProcessedBatchPenalty * emptyBatches
		}
	} else {
		// Boost peers that have never been selected.
		return s.BlockProviderMaxScore()
	}
	return math.Round(score*10000) / 10000
}

// BlockProviderStartScore exposes block provider's start score.
func (s *PeerScorer) BlockProviderStartScore() float64 {
	return blockProviderStartScore
}

// IncrementRequestedBlocks increments the number of blocks that have been requested from peer.
func (s *PeerScorer) IncrementRequestedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].requestedBlocks += cnt
}

// RequestedBlocks returns number of blocks requested from a peer.
func (s *PeerScorer) RequestedBlocks(pid peer.ID) uint64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.requestedBlocks(pid)
}

// requestedBlocks is a lock-free version of RequestedBlocks.
func (s *PeerScorer) requestedBlocks(pid peer.ID) uint64 {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.requestedBlocks
	}
	return 0
}

// IncrementReturnedBlocks increments the number of blocks that have been returned by peer.
func (s *PeerScorer) IncrementReturnedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].returnedBlocks += cnt
}

// ReturnedBlocks returns number of blocks returned by a peer.
func (s *PeerScorer) ReturnedBlocks(pid peer.ID) uint64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.returnedBlocks(pid)
}

// returnedBlocks is a lock-free version of ReturnedBlocks.
func (s *PeerScorer) returnedBlocks(pid peer.ID) uint64 {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.returnedBlocks
	}
	return 0
}

// IncrementProcessedBlocks increments the number of blocks that have been successfully processed.
func (s *PeerScorer) IncrementProcessedBlocks(pid peer.ID, cnt uint64) {
	s.store.Lock()
	defer s.store.Unlock()

	if _, ok := s.store.peers[pid]; !ok {
		s.store.peers[pid] = &peerData{}
	}
	s.store.peers[pid].processedBlocks += cnt
}

// ProcessedBlocks returns number of peer returned blocks that are successfully processed.
func (s *PeerScorer) ProcessedBlocks(pid peer.ID) uint64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.processedBlocks(pid)
}

// processedBlocks is a lock-free version of ProcessedBlocks.
func (s *PeerScorer) processedBlocks(pid peer.ID) uint64 {
	if peerData, ok := s.store.peers[pid]; ok {
		return peerData.processedBlocks
	}
	return 0
}

// DecayBlockProvidersStats updates block provider counters by decaying them.
// This urges peers to keep up the performance to get a high score (and allows new peers to contest previously high
// scoring ones).
func (s *PeerScorer) DecayBlockProvidersStats() {
	s.store.Lock()
	defer s.store.Unlock()

	for _, peerData := range s.store.peers {
		peerData.requestedBlocks = uint64(math.Round(float64(peerData.requestedBlocks) * s.config.BlockProviderDecay))
		// Once requested blocks stats drops to the half of batch size, reset stats.
		if peerData.requestedBlocks < uint64(flags.Get().BlockBatchLimit/2) {
			peerData.requestedBlocks = 0
			peerData.returnedBlocks = 0
			peerData.processedBlocks = 0
			continue
		}
		peerData.returnedBlocks = uint64(math.Round(float64(peerData.returnedBlocks) * s.config.BlockProviderDecay))
		peerData.processedBlocks = uint64(math.Round(float64(peerData.processedBlocks) * s.config.BlockProviderDecay))
	}
}

// SortBlockProviders returns list of block providers sorted by score in descending order.
func (s *PeerScorer) SortBlockProviders(pids []peer.ID) []peer.ID {
	s.store.Lock()
	defer s.store.Unlock()

	if len(pids) == 0 {
		return pids
	}
	scores := make(map[peer.ID]float64, len(pids))
	peers := make([]peer.ID, len(pids))
	for i, pid := range pids {
		scores[pid] = s.scoreBlockProvider(pid)
		peers[i] = pid
	}
	sort.SliceStable(peers, func(i, j int) bool {
		return scores[peers[i]] > scores[peers[j]]
	})
	return peers
}

// BlockProviderScorePretty returns full scoring information about a given peer.
func (s *PeerScorer) BlockProviderScorePretty(pid peer.ID) string {
	s.store.Lock()
	defer s.store.Unlock()
	score := s.scoreBlockProvider(pid)
	return fmt.Sprintf("[%0.2f%%, raw: %v,  req: %d, ret: %d, proc: %d]",
		(score/s.BlockProviderMaxScore())*100, score,
		s.requestedBlocks(pid), s.returnedBlocks(pid), s.processedBlocks(pid))
}

// BlockProviderMaxScore exposes maximum score attainable by peers.
func (s *PeerScorer) BlockProviderMaxScore() float64 {
	return s.BlockProviderStartScore() +
		s.config.BlockProviderReturnedBlocksWeight +
		s.config.BlockProviderProcessedBlocksWeight
}
