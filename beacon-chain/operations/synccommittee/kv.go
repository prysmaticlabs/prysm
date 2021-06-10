package synccommittee

import (
	"sync"

	"github.com/hashicorp/vault/sdk/queue"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var hashFn = hashutil.HashProto

// Store defines the caches for various sync committee objects
// such as signature(un-aggregated) and contribution(aggregated).
type Store struct {
	signatureLock     sync.RWMutex
	signatureCache    *queue.PriorityQueue
	contributionLock  sync.RWMutex
	contributionCache *queue.PriorityQueue
}

// NewStore initializes a new sync committee store.
func NewStore() *Store {
	return &Store{
		signatureCache:    queue.New(),
		contributionCache: queue.New(),
	}
}
