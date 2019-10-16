// Package peerstatus is a threadsafe global cache to store recent peer status messages for access
// across multiple services.
package peerstatus

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

var lock sync.RWMutex
var peerStatuses = make(map[peer.ID]*pb.Status)
var lastUpdated = make(map[peer.ID]time.Time)

// Get most recent status from peer in cache. Threadsafe.
func Get(pid peer.ID) *pb.Status {
	lock.RLock()
	defer lock.RUnlock()
	return peerStatuses[pid]
}

// Set most recent status from peer in cache. Threadsafe.
func Set(pid peer.ID, status *pb.Status) {
	lock.Lock()
	defer lock.Unlock()
	peerStatuses[pid] = status
	lastUpdated[pid] = roughtime.Now()
}

// Delete peer status from cache. Threadsafe.
func Delete(pid peer.ID) {
	lock.Lock()
	defer lock.Unlock()
	delete(peerStatuses, pid)
	delete(lastUpdated, pid)
}

// Count of peer statuses in cache. Threadsafe.
func Count() int {
	lock.RLock()
	defer lock.RUnlock()
	return len(peerStatuses)
}

// Keys is the list of peer IDs which status exists. Threadsafe.
func Keys() []peer.ID {
	lock.RLock()
	defer lock.RUnlock()
	keys := make([]peer.ID, 0, Count())
	for k := range peerStatuses {
		keys = append(keys, k)
	}
	return keys
}

// LastUpdated time which the status was set for the given peer. Threadsafe.
func LastUpdated(pid peer.ID) time.Time {
	lock.RLock()
	defer lock.RUnlock()
	return lastUpdated[pid]
}

// Clear the cache. This method should only be used for tests.
func Clear() {
	peerStatuses = make(map[peer.ID]*pb.Status)
}
