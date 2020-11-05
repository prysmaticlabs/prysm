package cache

import (
	"sync"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// StateSummaryCache caches state summary object.
type StateSummaryCache struct {
	initSyncStateSummaries     map[[32]byte]*pb.StateSummary
	initSyncStateSummariesLock sync.RWMutex
}

// NewStateSummaryCache creates a new state summary cache.
func NewStateSummaryCache() *StateSummaryCache {
	return &StateSummaryCache{
		initSyncStateSummaries: make(map[[32]byte]*pb.StateSummary),
	}
}

// Put saves a state summary to the initial sync state summaries cache.
func (s *StateSummaryCache) Put(r [32]byte, b *pb.StateSummary) {
	s.initSyncStateSummariesLock.Lock()
	defer s.initSyncStateSummariesLock.Unlock()
	s.initSyncStateSummaries[r] = b
}

// Has checks if a state summary exists in the initial sync state summaries cache using the root
// of the block.
func (s *StateSummaryCache) Has(r [32]byte) bool {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()
	_, ok := s.initSyncStateSummaries[r]
	return ok
}

// Get retrieves a state summary from the initial sync state summaries cache using the root of
// the block.
func (s *StateSummaryCache) Get(r [32]byte) *pb.StateSummary {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()
	b := s.initSyncStateSummaries[r]
	return b
}

// GetAll retrieves all the beacon state summaries from the initial sync state summaries cache, the returned
// state summaries are unordered.
func (s *StateSummaryCache) GetAll() []*pb.StateSummary {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()

	summaries := make([]*pb.StateSummary, 0, len(s.initSyncStateSummaries))
	for _, b := range s.initSyncStateSummaries {
		summaries = append(summaries, b)
	}
	return summaries
}

// Clear clears out the initial sync state summaries cache.
func (s *StateSummaryCache) Clear() {
	s.initSyncStateSummariesLock.Lock()
	defer s.initSyncStateSummariesLock.Unlock()
	s.initSyncStateSummaries = make(map[[32]byte]*pb.StateSummary)
}
