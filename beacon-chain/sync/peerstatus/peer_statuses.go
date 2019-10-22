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
var failureCount = make(map[peer.ID]int)
var maxFailureThreshold = 3

type peerStatus struct {
	status      *pb.Status
	lastUpdated time.Time
}

// Get most recent status from peer in cache. Threadsafe.
func Get(pid peer.ID) *pb.Status {
	lock.RLock()
	defer lock.RUnlock()
	if pStatus, ok := peerStatuses[pid]; ok {
		return pStatus.status
	}
	return nil
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
	if pStatus, ok := peerStatuses[pid]; ok {
		return pStatus.lastUpdated
	}
	return time.Unix(0, 0)
}

// IncreaseFailureCount increases the failure count for the particular peer.
func IncreaseFailureCount(pid peer.ID) {
	lock.Lock()
	defer lock.Unlock()
	count, ok := failureCount[pid]
	if !ok {
		return
	}
	count++
	failureCount[pid] = count
}

// FailureCount returns the failure count for the particular peer.
func FailureCount(pid peer.ID) int {
	lock.RLock()
	defer lock.RUnlock()
	count, ok := failureCount[pid]
	if !ok {
		return 0
	}
	return count
}

// IsBadPeer checks whether the given peer has
// exceeded the number of bad handshakes threshold.
func IsBadPeer(pid peer.ID) bool {
	lock.RLock()
	defer lock.RUnlock()
	count, ok := failureCount[pid]
	if !ok {
		return false
	}
	return count > maxFailureThreshold
}

// Clear the cache. This method should only be used for tests.
func Clear() {
	peerStatuses = make(map[peer.ID]*peerStatus)
	failureCount = make(map[peer.ID]int)
}
