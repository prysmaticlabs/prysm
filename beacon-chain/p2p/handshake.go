package p2p

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var handshakes = make(map[peer.ID]*pb.Hello)
var handshakeLock sync.Mutex

// AddHandshake to the local records for initial sync.
func (p *Service) AddHandshake(pid peer.ID, hello *pb.Hello) {
	handshakeLock.Lock()
	defer handshakeLock.Unlock()
	handshakes[pid] = hello
}

// Handshakes has not been implemented yet and it may be moved to regular sync...
func (p *Service) Handshakes() map[peer.ID]*pb.Hello {
	return nil
}
