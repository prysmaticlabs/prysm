// Package peers provides information about peers at the eth2 protocol level.
// "Protocol level" is the level above the network level, so this layer never sees or interacts with (for example) hosts that are
// uncontactable due to being down, firewalled, etc.  Instead, this works with peers that are contactable but may or may not be of
// the correct fork version, not currently required due to the number of current connections, etc.
//
// A peer can have one of a number of states:
//
// - connected if we are able to talk to the remote peer
// - connecting if we are attempting to be able to talk to the remote peer
// - disconnecting if we are attempting to stop being able to talk to the remote peer
// - disconnected if we are not able to talk to the remote peer
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
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// PeerConnectionState is the state of the connection.
type PeerConnectionState ethpb.ConnectionState

const (
	// PeerDisconnected means there is no connection to the peer.
	PeerDisconnected PeerConnectionState = iota
	// PeerDisconnecting means there is an on-going attempt to disconnect from the peer.
	PeerDisconnecting
	// PeerConnected means the peer has an active connection.
	PeerConnected
	// PeerConnecting means there is an on-going attempt to connect to the peer.
	PeerConnecting
)

// Additional buffer beyond current peer limit, from which we can store
// the relevant peer statuses.
const maxLimitBuffer = 150

var (
	// ErrPeerUnknown is returned when there is an attempt to obtain data from a peer that is not known.
	ErrPeerUnknown = errors.New("peer unknown")
)

// Status is the structure holding the peer status information.
type Status struct {
	lock     sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	status   map[peer.ID]*peerStatus
	maxLimit int
	scorer   *PeerScorer
}

// peerStatus is the status of an individual peer at the protocol level.
type peerStatus struct {
	address               ma.Multiaddr
	direction             network.Direction
	peerState             PeerConnectionState
	chainState            *pb.Status
	enr                   *enr.Record
	metaData              *pb.MetaData
	chainStateLastUpdated time.Time
}

// NewStatus creates a new status entity.
func NewStatus(maxBadResponses int, peerLimit int) *Status {
	ctx, cancel := context.WithCancel(context.Background())
	return &Status{
		ctx:      ctx,
		cancel:   cancel,
		status:   make(map[peer.ID]*peerStatus),
		maxLimit: maxLimitBuffer + peerLimit,
		scorer: NewPeerScorer(ctx, &PeerScorerParams{
			BadResponsesThreshold:     maxBadResponses,
			BadResponsesWeight:        -100,
			BadResponsesDecayInterval: time.Hour,
		}),
	}
}

// Scorer exposes peer scoring service.
func (p *Status) Scorer() *PeerScorer {
	return p.scorer
}

// ScorePeer returns calculated application-level peer scorer.
func (p *Status) ScorePeer(pid peer.ID) float64 {
	return p.scorer.Score(pid)
}

// MaxPeerLimit returns the max peer limit stored in
// the current peer store.
func (p *Status) MaxPeerLimit() int {
	return p.maxLimit
}

// Add adds a peer.
// If a peer already exists with this ID its address and direction are updated with the supplied data.
func (p *Status) Add(record *enr.Record, pid peer.ID, address ma.Multiaddr, direction network.Direction) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if status, ok := p.status[pid]; ok {
		// Peer already exists, just update its address info.
		status.address = address
		status.direction = direction
		if record != nil {
			status.enr = record
		}
		return
	}
	status := &peerStatus{
		address:   address,
		direction: direction,
		// Peers start disconnected; state will be updated when the handshake process begins.
		peerState: PeerDisconnected,
	}
	if record != nil {
		status.enr = record
	}
	p.status[pid] = status
	p.scorer.AddPeer(pid)
}

// Address returns the multiaddress of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) Address(pid peer.ID) (ma.Multiaddr, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.address, nil
	}
	return nil, ErrPeerUnknown
}

// Direction returns the direction of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) Direction(pid peer.ID) (network.Direction, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.direction, nil
	}
	return network.DirUnknown, ErrPeerUnknown
}

// ENR returns the enr for the corresponding peer id.
func (p *Status) ENR(pid peer.ID) (*enr.Record, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.enr, nil
	}
	return nil, ErrPeerUnknown
}

// SetChainState sets the chain state of the given remote peer.
func (p *Status) SetChainState(pid peer.ID, chainState *pb.Status) {
	p.lock.Lock()
	defer p.lock.Unlock()

	status := p.fetch(pid)
	status.chainState = chainState
	status.chainStateLastUpdated = roughtime.Now()
}

// ChainState gets the chain state of the given remote peer.
// This can return nil if there is no known chain state for the peer.
// This will error if the peer does not exist.
func (p *Status) ChainState(pid peer.ID) (*pb.Status, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.chainState, nil
	}
	return nil, ErrPeerUnknown
}

// IsActive checks if a peers is active and returns the result appropriately.
func (p *Status) IsActive(pid peer.ID) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	status, ok := p.status[pid]
	return ok && (status.peerState == PeerConnected || status.peerState == PeerConnecting)
}

// SetMetadata sets the metadata of the given remote peer.
func (p *Status) SetMetadata(pid peer.ID, metaData *pb.MetaData) {
	p.lock.Lock()
	defer p.lock.Unlock()

	status := p.fetch(pid)
	status.metaData = metaData
}

// Metadata returns a copy of the metadata corresponding to the provided
// peer id.
func (p *Status) Metadata(pid peer.ID) (*pb.MetaData, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return proto.Clone(status.metaData).(*pb.MetaData), nil
	}
	return nil, ErrPeerUnknown
}

// CommitteeIndices retrieves the committee subnets the peer is subscribed to.
func (p *Status) CommitteeIndices(pid peer.ID) ([]uint64, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		if status.enr == nil || status.metaData == nil {
			return []uint64{}, nil
		}
		return retrieveIndicesFromBitfield(status.metaData.Attnets), nil
	}
	return nil, ErrPeerUnknown
}

// SubscribedToSubnet retrieves the peers subscribed to the given
// committee subnet.
func (p *Status) SubscribedToSubnet(index uint64) []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()

	peers := make([]peer.ID, 0)
	for pid, status := range p.status {
		// look at active peers
		connectedStatus := status.peerState == PeerConnecting || status.peerState == PeerConnected
		if connectedStatus && status.metaData != nil && status.metaData.Attnets != nil {
			indices := retrieveIndicesFromBitfield(status.metaData.Attnets)
			for _, idx := range indices {
				if idx == index {
					peers = append(peers, pid)
					break
				}
			}
		}
	}
	return peers
}

// SetConnectionState sets the connection state of the given remote peer.
func (p *Status) SetConnectionState(pid peer.ID, state PeerConnectionState) {
	p.lock.Lock()
	defer p.lock.Unlock()

	status := p.fetch(pid)
	status.peerState = state
}

// ConnectionState gets the connection state of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) ConnectionState(pid peer.ID) (PeerConnectionState, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.peerState, nil
	}
	return PeerDisconnected, ErrPeerUnknown
}

// ChainStateLastUpdated gets the last time the chain state of the given remote peer was updated.
// This will error if the peer does not exist.
func (p *Status) ChainStateLastUpdated(pid peer.ID) (time.Time, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if status, ok := p.status[pid]; ok {
		return status.chainStateLastUpdated, nil
	}
	return roughtime.Now(), ErrPeerUnknown
}

// IsBad states if the peer is to be considered bad.
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (p *Status) IsBad(pid peer.ID) bool {
	return p.scorer.IsBadPeer(pid)
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

// Bad returns the peers that are bad.
func (p *Status) Bad() []peer.ID {
	return p.scorer.BadPeers()
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

// Prune clears out and removes outdated and disconnected peers.
func (p *Status) Prune() {
	p.lock.Lock()
	defer p.lock.Unlock()

	currSize := len(p.status)
	// Exit early if there is nothing to prune.
	if currSize <= p.maxLimit {
		return
	}

	type peerResp struct {
		pid     peer.ID
		badResp int
	}
	peersToPrune := make([]*peerResp, 0)
	// Select disconnected peers with a smaller bad response count.
	for pid, status := range p.status {
		if status.peerState == PeerDisconnected && p.scorer.IsGoodPeer(pid) {
			badResp, err := p.scorer.BadResponses(pid)
			if err != nil {
				badResp = 0
			}
			peersToPrune = append(peersToPrune, &peerResp{
				pid:     pid,
				badResp: badResp,
			})
		}
	}

	// Sort peers in ascending order, so the peers with the
	// least amount of bad responses are pruned first. This
	// is to protect the node from malicious/lousy peers so
	// that their memory is still kept.
	sort.Slice(peersToPrune, func(i, j int) bool {
		return peersToPrune[i].badResp < peersToPrune[j].badResp
	})

	limitDiff := currSize - p.maxLimit
	if limitDiff > len(peersToPrune) {
		limitDiff = len(peersToPrune)
	}

	peersToPrune = peersToPrune[:limitDiff]

	// Delete peers from map.
	for _, peerRes := range peersToPrune {
		delete(p.status, peerRes.pid)
		p.scorer.RemovePeer(peerRes.pid)
	}
}

// BestFinalized returns the highest finalized epoch equal to or higher than ours that is agreed upon by the majority of peers.
// This method may not return the absolute highest finalized, but the finalized epoch in which most peers can serve blocks.
// Ideally, all peers would be reporting the same finalized epoch but some may be behind due to their own latency, or because of
// their finalized epoch at the time we queried them.
// Returns the best finalized root, epoch number, and list of peers that are at or beyond that epoch.
func (p *Status) BestFinalized(maxPeers int, ourFinalizedEpoch uint64) (uint64, []peer.ID) {
	connected := p.Connected()
	finalizedEpochVotes := make(map[uint64]uint64)
	pidEpoch := make(map[peer.ID]uint64, len(connected))
	potentialPIDs := make([]peer.ID, 0, len(connected))
	for _, pid := range connected {
		peerChainState, err := p.ChainState(pid)
		if err == nil && peerChainState != nil && peerChainState.FinalizedEpoch >= ourFinalizedEpoch {
			finalizedEpochVotes[peerChainState.FinalizedEpoch]++
			pidEpoch[pid] = peerChainState.FinalizedEpoch
			potentialPIDs = append(potentialPIDs, pid)
		}
	}

	// Select the target epoch, which is the epoch most peers agree upon.
	var targetEpoch uint64
	var mostVotes uint64
	for epoch, count := range finalizedEpochVotes {
		if count > mostVotes {
			mostVotes = count
			targetEpoch = epoch
		}
	}

	// Sort PIDs by finalized epoch, in decreasing order.
	sort.Slice(potentialPIDs, func(i, j int) bool {
		return pidEpoch[potentialPIDs[i]] > pidEpoch[potentialPIDs[j]]
	})

	// Trim potential peers to those on or after target epoch.
	for i, pid := range potentialPIDs {
		if pidEpoch[pid] < targetEpoch {
			potentialPIDs = potentialPIDs[:i]
			break
		}
	}

	// Trim potential peers to at most maxPeers.
	if len(potentialPIDs) > maxPeers {
		potentialPIDs = potentialPIDs[:maxPeers]
	}

	return targetEpoch, potentialPIDs
}

// fetch is a helper function that fetches a peer status, possibly creating it.
func (p *Status) fetch(pid peer.ID) *peerStatus {
	if _, ok := p.status[pid]; !ok {
		p.status[pid] = &peerStatus{}
	}
	return p.status[pid]
}

// HighestEpoch returns the highest epoch reported epoch amongst peers.
func (p *Status) HighestEpoch() uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()
	var highestSlot uint64
	for _, ps := range p.status {
		if ps != nil && ps.chainState != nil && ps.chainState.HeadSlot > highestSlot {
			highestSlot = ps.chainState.HeadSlot
		}
	}
	return helpers.SlotToEpoch(highestSlot)
}

func retrieveIndicesFromBitfield(bitV bitfield.Bitvector64) []uint64 {
	committeeIdxs := make([]uint64, 0, bitV.Count())
	for i := uint64(0); i < 64; i++ {
		if bitV.BitAt(i) {
			committeeIdxs = append(committeeIdxs, i)
		}
	}
	return committeeIdxs
}
