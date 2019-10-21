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
var peerStatuses = make(map[peer.ID]*peerStatus)
var failureCount = make(map[peer.ID]uint64)

type peerStatus struct {
	status      *pb.Status
	lastUpdated time.Time
}

// Get most recent status from peer in cache. Threadsafe.
func Get(pid peer.ID) *pb.Status {
	lock.RLock()
	defer lock.RUnlock()
	return peerStatuses[pid].status
}

// Set most recent status from peer in cache. Threadsafe.
func Set(pid peer.ID, status *pb.Status) {
	lock.Lock()
	defer lock.Unlock()
	if pStatus, ok := peerStatuses[pid]; ok {
		pStatus.status = status
		peerStatuses[pid] = pStatus
		return
	}
	peerStatuses[pid] = &peerStatus{
		status:      status,
		lastUpdated: roughtime.Now(),
	}
	failureCount[pid] = 0
}

// Delete peer status from cache. Threadsafe.
func Delete(pid peer.ID) {
	lock.Lock()
	defer lock.Unlock()
	delete(peerStatuses, pid)
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
	keys := make([]peer.ID, 0, len(peerStatuses))
	for k := range peerStatuses {
		keys = append(keys, k)
	}
	return keys
}

// LastUpdated time which the status was set for the given peer. Threadsafe.
func LastUpdated(pid peer.ID) time.Time {
	lock.RLock()
	defer lock.RUnlock()
	return peerStatuses[pid].lastUpdated
}

// BumpFailureCount increases the failure count for the particular peer.
func BumpFailureCount(pid peer.ID) {
	count, ok := failureCount[pid]
	if !ok {
		return
	}
	count++
	failureCount[pid] = count
}

// FailureCount returns the failure count for the particular peer.
func FailureCount(pid peer.ID) uint64 {
	count, ok := failureCount[pid]
	if !ok {
		return 0
	}
	return count
}

// Clear the cache. This method should only be used for tests.
func Clear() {
	peerStatuses = make(map[peer.ID]*peerStatus)
}
