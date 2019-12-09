// Package peers provides information about peers at the Ethereum protocol level.
// "Protocol level" is the level above the network level, so this layer never sees or interacts with (for example) hosts that are
// uncontactable due to being down, firewalled, etc.  Instead, this works with peers that are contactable but may or may not be of
// the correct fork version, not currently required due to the number of current connections, etc.
//
// A peer can have one of a number of states:
//
// - disconnected if we are not able to talk to the remote peer
// - connecting if we are attempting to be able to talk to the remote peer
// - connected if we are able to talk to the remote peer
// - disconnecting if we are attempting to stop being able to talk to the remote peer
//
// For convenience, there are two aggregate states expressed in functions:
//
// - active if we are connecting or connected
// - inactive if we are disconnecting or disconnected
//
// Peer information is persistent for the run of the service.  This allows for collection of useful long-term statistics such as
// number of bad responses obtained from the peer, giving the basis for decisions to not talk to known-bad peers.
package peers

import (
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// PeerConnectionState is the state of the connection.
type PeerConnectionState int

const (
	// PeerDisconnected means there is no connection to the peer.
	PeerDisconnected PeerConnectionState = iota
	// PeerConnecting means there is an on-going attempt to connect to the peer.
	PeerConnecting
	// PeerConnected means the peer has an active connection.
	PeerConnected
	// PeerDisconnecting means there is an on-going attempt to disconnect from the peer.
	PeerDisconnecting
)

// Status is the structure holding the peer status information.
type Status struct {
	lock            sync.RWMutex
	maxBadResponses int
	status          map[peer.ID]*peerStatus
}

// peerStatus is the status of an individual peer at the protocol level.
type peerStatus struct {
	address               ma.Multiaddr
	direction             network.Direction
	peerState             PeerConnectionState
	chainState            *pb.Status
	chainStateLastUpdated time.Time
	badResponses          int
}

// NewStatus creates a new status entity.
func NewStatus(maxBadResponses int) *Status {
	return &Status{
		maxBadResponses: maxBadResponses,
		status:          make(map[peer.ID]*peerStatus),
	}
}

// MaxBadResponses returns the maximum number of bad responses a peer can provide before it is considered bad.
func (p *Status) MaxBadResponses() int {
	return p.maxBadResponses
}

// Add adds a peer.  This will error if the peer already exists.
func (p *Status) Add(pid peer.ID, address ma.Multiaddr, direction network.Direction) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, ok := p.status[pid]; ok {
		return fmt.Errorf("peer %v already exists", pid)
	}

	status := &peerStatus{
		address:   address,
		direction: direction,
		peerState: PeerDisconnected,
	}

	p.status[pid] = status
	return nil
}

// Address returns the multiaddress of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) Address(pid peer.ID) (ma.Multiaddr, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.address, nil
	}
	return nil, fmt.Errorf("peer %v unknown", pid)
}

// Direction returns the direction of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) Direction(pid peer.ID) (network.Direction, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.direction, nil
	}
	return network.DirUnknown, fmt.Errorf("peer %v unknown", pid)
}

// SetChainState sets the chain state of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) SetChainState(pid peer.ID, chainState *pb.Status) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if status, ok := p.status[pid]; ok {
		status.chainState = chainState
		return nil
	}
	return fmt.Errorf("peer %v unknown", pid)
}

// ChainState gets the chain state of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) ChainState(pid peer.ID) (*pb.Status, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		if status.chainState == nil {
			return nil, fmt.Errorf("peer %v has no known chain state", pid)
		}
		return status.chainState, nil
	}
	return nil, fmt.Errorf("peer %v unknown", pid)
}

// SetConnectionState sets the connection state of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) SetConnectionState(pid peer.ID, state PeerConnectionState) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if status, ok := p.status[pid]; ok {
		status.peerState = state
		status.chainStateLastUpdated = roughtime.Now()
		return nil
	}
	return fmt.Errorf("peer %v unknown", pid)
}

// ConnectionState gets the connection state of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) ConnectionState(pid peer.ID) (PeerConnectionState, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.peerState, nil
	}
	return PeerDisconnected, fmt.Errorf("peer %v unknown", pid)
}

// ChainStateLastUpdated gets the last time the chain state of the given remote peer was updated.
// This will error if the peer does not exist.
func (p *Status) ChainStateLastUpdated(pid peer.ID) (time.Time, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.chainStateLastUpdated, nil
	}
	return time.Now(), fmt.Errorf("peer %v unknown", pid)
}

// IncrementBadResponses increments the number of bad responses we have received from the given remote peer.
// This will error if the peer does not exist.
func (p *Status) IncrementBadResponses(pid peer.ID) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if status, ok := p.status[pid]; ok {
		status.badResponses++
		return nil
	}
	return fmt.Errorf("peer %v unknown", pid)
}

// BadResponses obtains the number of bad responses we have received from the given remote peer.
// This will error if the peer does not exist.
func (p *Status) BadResponses(pid peer.ID) (int, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.badResponses, nil
	}
	return -1, fmt.Errorf("peer %v unknown", pid)
}

// IsBad states if the peer is to be considered bad.
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (p *Status) IsBad(pid peer.ID) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.badResponses >= p.maxBadResponses
	}
	return false
}

// Connecting returns the peers that are connecting.
func (p *Status) Connecting() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		if status.peerState == PeerConnecting {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Connected returns the peers that are connected.
func (p *Status) Connected() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		if status.peerState == PeerConnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Active returns the peers that are connecting or connected.
func (p *Status) Active() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		if status.peerState == PeerConnecting || status.peerState == PeerConnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Disconnecting returns the peers that are disconnecting.
func (p *Status) Disconnecting() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		if status.peerState == PeerDisconnecting {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Disconnected returns the peers that are disconnected.
func (p *Status) Disconnected() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		if status.peerState == PeerDisconnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Inactive returns the peers that are disconnecting or disconnected.
func (p *Status) Inactive() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		if status.peerState == PeerDisconnecting || status.peerState == PeerDisconnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// All returns all the peers regardless of state.
func (p *Status) All() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pids := make([]peer.ID, 0, len(p.status))
	for pid := range p.status {
		pids = append(pids, pid)
	}
	return pids
}
