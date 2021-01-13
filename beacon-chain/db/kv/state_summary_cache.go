package kv

import (
	"sync"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

const stateSummaryCachePruneCount = 128

// stateSummaryCache caches state summary object.
type stateSummaryCache struct {
	initSyncStateSummaries     map[[32]byte]*pb.StateSummary
	initSyncStateSummariesLock sync.RWMutex
}

// newStateSummaryCache creates a new state summary cache.
func newStateSummaryCache() *stateSummaryCache {
	return &stateSummaryCache{
		initSyncStateSummaries: make(map[[32]byte]*pb.StateSummary),
	}
}

// put saves a state summary to the initial sync state summaries cache.
func (s *stateSummaryCache) put(r [32]byte, b *pb.StateSummary) {
	s.initSyncStateSummariesLock.Lock()
	defer s.initSyncStateSummariesLock.Unlock()
	s.initSyncStateSummaries[r] = b
}

// has checks if a state summary exists in the initial sync state summaries cache using the root
// of the block.
func (s *stateSummaryCache) has(r [32]byte) bool {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()
	_, ok := s.initSyncStateSummaries[r]
	return ok
}

// get retrieves a state summary from the initial sync state summaries cache using the root of
// the block.
func (s *stateSummaryCache) get(r [32]byte) *pb.StateSummary {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()
	b := s.initSyncStateSummaries[r]
	return b
}

// len retrieves the state summary count from the state summaries cache.
func (s *stateSummaryCache) len() int {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()
	return len(s.initSyncStateSummaries)
}

// GetAll retrieves all the beacon state summaries from the initial sync state summaries cache, the returned
// state summaries are unordered.
func (s *stateSummaryCache) getAll() []*pb.StateSummary {
	s.initSyncStateSummariesLock.RLock()
	defer s.initSyncStateSummariesLock.RUnlock()

	summaries := make([]*pb.StateSummary, 0, len(s.initSyncStateSummaries))
	for _, b := range s.initSyncStateSummaries {
		summaries = append(summaries, b)
	}
	return summaries
}

// Clear clears out the initial sync state summaries cache.
func (s *stateSummaryCache) clear() {
	s.initSyncStateSummariesLock.Lock()
	defer s.initSyncStateSummariesLock.Unlock()
	s.initSyncStateSummaries = make(map[[32]byte]*pb.StateSummary)
}
