// Package peerstatus is a threadsafe global cache to store recent peer status messages for access
// across multiple services.
package peerstatus

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var lock sync.RWMutex
var peerStatuses = make(map[peer.ID]*pb.Hello)

// Get most recent status from peer in cache.
func Get(pid peer.ID) *pb.Hello {
	lock.RLock()
	defer lock.RUnlock()
	return peerStatuses[pid]
}

// Set most recent status from peer in cache.
func Set(pid peer.ID, status *pb.Hello) {
	lock.Lock()
	defer lock.Unlock()
	peerStatuses[pid] = status
}

func Delete(pid peer.ID) {
	lock.Lock()
	defer lock.Unlock()
	delete(peerStatuses, pid)
}

func Count() int {
	return len(peerStatuses)
}

func Keys() []peer.ID {
	lock.RLock()
	defer lock.RUnlock()
	keys := make([]peer.ID, 0, Count())
	for k := range peerStatuses {
		keys = append(keys, k)
	}
	return keys
}

func Clear() {
	peerStatuses = make(map[peer.ID]*pb.Hello)
}