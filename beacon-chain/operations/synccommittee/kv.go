package synccommittee

import (
	"sync"

	"github.com/prysmaticlabs/prysm/v3/container/queue"
)

// Store defines the caches for various sync committee objects
// such as message(un-aggregated) and contribution(aggregated).
type Store struct {
	messageCache      *queue.PriorityQueue
	contributionCache *queue.PriorityQueue
	messageLock       sync.RWMutex
	contributionLock  sync.RWMutex
}

// NewStore initializes a new sync committee store.
func NewStore() *Store {
	return &Store{
		messageCache:      queue.New(),
		contributionCache: queue.New(),
	}
}
