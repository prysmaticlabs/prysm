package peers

import (
	"math"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
)

// ScoreBlockProvider calculates and returns total score based on returned and processed blocks.
func (s *PeerScorer) ScoreBlockProvider(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.scoreBlockProvider(pid)
}

// scoreBlockProvider is a lock-free version of ScoreBlockProvider.
func (s *PeerScorer) scoreBlockProvider(pid peer.ID) float64 {
	score := float64(0)
	peerData, ok := s.store.peers[pid]
	if !ok {
		return score
	}
	if peerData.requestedBlocks > 0 {
		// Score returned/requested ratio. If no blocks has been returned, apply as a penalty.
		if peerData.returnedBlocks == 0 {
			score += -1.0 * s.config.BlockProviderReturnedBlocksWeight
		} else {
			returnedBlocksScore := float64(peerData.returnedBlocks) / float64(peerData.requestedBlocks)
			returnedBlocksScore = returnedBlocksScore * s.config.BlockProviderReturnedBlocksWeight
			score += returnedBlocksScore
		}
		// Score processed/requested ratio. If no blocks has been processed, apply as a penalty.
		if peerData.processedBlocks == 0 {
			score += -1.0 * s.config.BlockProviderProcessedBlocksWeight
		} else {
			processedBlocksScore := float64(peerData.processedBlocks) / float64(peerData.requestedBlocks)
			processedBlocksScore = processedBlocksScore * s.config.BlockProviderProcessedBlocksWeight
			score += processedBlocksScore
		}
	}
	return math.Round(score*10000) / 10000
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
