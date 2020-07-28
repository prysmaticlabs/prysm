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

// Additional buffer beyond current peer limit, from which we can store the relevant peer statuses.
const maxLimitBuffer = 150

var (
	// ErrPeerUnknown is returned when there is an attempt to obtain data from a peer that is not known.
	ErrPeerUnknown = errors.New("peer unknown")
)

// Status is the structure holding the peer status information.
type Status struct {
	ctx    context.Context
	scorer *PeerScorer
	store  *peerDataStore
}

// StatusConfig represents peer status service params.
type StatusConfig struct {
	// PeerLimit specifies maximum amount of concurrent peers that are expected to be connect to the node.
	PeerLimit int
	// ScorerParams holds peer scorer configuration params.
	ScorerParams *PeerScorerConfig
}

// NewStatus creates a new status entity.
func NewStatus(ctx context.Context, config *StatusConfig) *Status {
	store := newPeerDataStore(ctx, &peerDataStoreConfig{
		maxPeers: maxLimitBuffer + config.PeerLimit,
	})
	return &Status{
		ctx:    ctx,
		store:  store,
		scorer: newPeerScorer(ctx, store, config.ScorerParams),
	}
}

// Scorer exposes peer scoring service.
func (p *Status) Scorer() *PeerScorer {
	return p.scorer
}

// MaxPeerLimit returns the max peer limit stored in the current peer store.
func (p *Status) MaxPeerLimit() int {
	return p.store.config.maxPeers
}

// Add adds a peer.
// If a peer already exists with this ID its address and direction are updated with the supplied data.
func (p *Status) Add(record *enr.Record, pid peer.ID, address ma.Multiaddr, direction network.Direction) {
	p.store.Lock()
	defer p.store.Unlock()

	if peerData, ok := p.store.peers[pid]; ok {
		// Peer already exists, just update its address info.
		peerData.address = address
		peerData.direction = direction
		if record != nil {
			peerData.enr = record
		}
		return
	}
	peerData := &peerData{
		address:   address,
		direction: direction,
		// Peers start disconnected; state will be updated when the handshake process begins.
		connState: PeerDisconnected,
	}
	if record != nil {
		peerData.enr = record
	}
	p.store.peers[pid] = peerData
}

// Address returns the multiaddress of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) Address(pid peer.ID) (ma.Multiaddr, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return peerData.address, nil
	}
	return nil, ErrPeerUnknown
}

// Direction returns the direction of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) Direction(pid peer.ID) (network.Direction, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return peerData.direction, nil
	}
	return network.DirUnknown, ErrPeerUnknown
}

// ENR returns the enr for the corresponding peer id.
func (p *Status) ENR(pid peer.ID) (*enr.Record, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return peerData.enr, nil
	}
	return nil, ErrPeerUnknown
}

// SetChainState sets the chain state of the given remote peer.
func (p *Status) SetChainState(pid peer.ID, chainState *pb.Status) {
	p.store.Lock()
	defer p.store.Unlock()

	peerData := p.fetch(pid)
	peerData.chainState = chainState
	peerData.chainStateLastUpdated = roughtime.Now()
}

// ChainState gets the chain state of the given remote peer.
// This can return nil if there is no known chain state for the peer.
// This will error if the peer does not exist.
func (p *Status) ChainState(pid peer.ID) (*pb.Status, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return peerData.chainState, nil
	}
	return nil, ErrPeerUnknown
}

// IsActive checks if a peers is active and returns the result appropriately.
func (p *Status) IsActive(pid peer.ID) bool {
	p.store.RLock()
	defer p.store.RUnlock()

	peerData, ok := p.store.peers[pid]
	return ok && (peerData.connState == PeerConnected || peerData.connState == PeerConnecting)
}

// SetMetadata sets the metadata of the given remote peer.
func (p *Status) SetMetadata(pid peer.ID, metaData *pb.MetaData) {
	p.store.Lock()
	defer p.store.Unlock()

	peerData := p.fetch(pid)
	peerData.metaData = metaData
}

// Metadata returns a copy of the metadata corresponding to the provided
// peer id.
func (p *Status) Metadata(pid peer.ID) (*pb.MetaData, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return proto.Clone(peerData.metaData).(*pb.MetaData), nil
	}
	return nil, ErrPeerUnknown
}

// CommitteeIndices retrieves the committee subnets the peer is subscribed to.
func (p *Status) CommitteeIndices(pid peer.ID) ([]uint64, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		if peerData.enr == nil || peerData.metaData == nil {
			return []uint64{}, nil
		}
		return retrieveIndicesFromBitfield(peerData.metaData.Attnets), nil
	}
	return nil, ErrPeerUnknown
}

// SubscribedToSubnet retrieves the peers subscribed to the given
// committee subnet.
func (p *Status) SubscribedToSubnet(index uint64) []peer.ID {
	p.store.RLock()
	defer p.store.RUnlock()

	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		// look at active peers
		connectedStatus := peerData.connState == PeerConnecting || peerData.connState == PeerConnected
		if connectedStatus && peerData.metaData != nil && peerData.metaData.Attnets != nil {
			indices := retrieveIndicesFromBitfield(peerData.metaData.Attnets)
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
	p.store.Lock()
	defer p.store.Unlock()

	peerData := p.fetch(pid)
	peerData.connState = state
}

// ConnectionState gets the connection state of the given remote peer.
// This will error if the peer does not exist.
func (p *Status) ConnectionState(pid peer.ID) (PeerConnectionState, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return peerData.connState, nil
	}
	return PeerDisconnected, ErrPeerUnknown
}

// ChainStateLastUpdated gets the last time the chain state of the given remote peer was updated.
// This will error if the peer does not exist.
func (p *Status) ChainStateLastUpdated(pid peer.ID) (time.Time, error) {
	p.store.RLock()
	defer p.store.RUnlock()

	if peerData, ok := p.store.peers[pid]; ok {
		return peerData.chainStateLastUpdated, nil
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
	p.store.RLock()
	defer p.store.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerConnecting {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Connected returns the peers that are connected.
func (p *Status) Connected() []peer.ID {
	p.store.RLock()
	defer p.store.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerConnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Active returns the peers that are connecting or connected.
func (p *Status) Active() []peer.ID {
	p.store.RLock()
	defer p.store.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerConnecting || peerData.connState == PeerConnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Disconnecting returns the peers that are disconnecting.
func (p *Status) Disconnecting() []peer.ID {
	p.store.RLock()
	defer p.store.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerDisconnecting {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Disconnected returns the peers that are disconnected.
func (p *Status) Disconnected() []peer.ID {
	p.store.RLock()
	defer p.store.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerDisconnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Inactive returns the peers that are disconnecting or disconnected.
func (p *Status) Inactive() []peer.ID {
	p.store.RLock()
	defer p.store.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerDisconnecting || peerData.connState == PeerDisconnected {
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
	p.store.RLock()
	defer p.store.RUnlock()
	pids := make([]peer.ID, 0, len(p.store.peers))
	for pid := range p.store.peers {
		pids = append(pids, pid)
	}
	return pids
}

// Prune clears out and removes outdated and disconnected peers.
func (p *Status) Prune() {
	p.store.Lock()
	defer p.store.Unlock()

	// Exit early if there is nothing to prune.
	if len(p.store.peers) <= p.store.config.maxPeers {
		return
	}

	type peerResp struct {
		pid     peer.ID
		badResp int
	}
	peersToPrune := make([]*peerResp, 0)
	// Select disconnected peers with a smaller bad response count.
	for pid, peerData := range p.store.peers {
		if peerData.connState == PeerDisconnected && !p.scorer.isBadPeer(pid) {
			peersToPrune = append(peersToPrune, &peerResp{
				pid:     pid,
				badResp: p.store.peers[pid].badResponses,
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

	limitDiff := len(p.store.peers) - p.store.config.maxPeers
	if limitDiff > len(peersToPrune) {
		limitDiff = len(peersToPrune)
	}

	peersToPrune = peersToPrune[:limitDiff]

	// Delete peers from map.
	for _, peerData := range peersToPrune {
		delete(p.store.peers, peerData.pid)
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
func (p *Status) fetch(pid peer.ID) *peerData {
	if _, ok := p.store.peers[pid]; !ok {
		p.store.peers[pid] = &peerData{}
	}
	return p.store.peers[pid]
}

// HighestEpoch returns the highest epoch reported epoch amongst peers.
func (p *Status) HighestEpoch() uint64 {
	p.store.RLock()
	defer p.store.RUnlock()
	var highestSlot uint64
	for _, peerData := range p.store.peers {
		if peerData != nil && peerData.chainState != nil && peerData.chainState.HeadSlot > highestSlot {
			highestSlot = peerData.chainState.HeadSlot
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
