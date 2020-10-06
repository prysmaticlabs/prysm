package data

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var (
	// ErrPeerUnknown is returned when there is an attempt to obtain data from a peer that is not known.
	ErrPeerUnknown = errors.New("peer unknown")
)

// StoreConfig holds peer store parameters.
type StoreConfig struct {
	MaxPeers int
}

// Store is a container for various peer related data (both protocol and app level).
// Container implements RWMutex, so data access can be restricted on the container level. This allows
// different components rely on the very same peer map container.
type Store struct {
	sync.RWMutex
	ctx    context.Context
	config *StoreConfig
	peers  map[peer.ID]*PeerData
}

// PeerData aggregates protocol and application level info about a single peer.
type PeerData struct {
	// Network related data.
	Address   ma.Multiaddr
	Direction network.Direction
	ConnState peers.PeerConnectionState
	Enr       *enr.Record
	// Chain related data.
	ChainState            *pb.Status
	MetaData              *pb.MetaData
	ChainStateLastUpdated time.Time
	// Scorers related data.
	BadResponses         int
	ProcessedBlocks      uint64
	BlockProviderUpdated time.Time
}

// NewStore creates new peer data store.
func NewStore(ctx context.Context, config *StoreConfig) *Store {
	return &Store{
		ctx:    ctx,
		config: config,
		peers:  make(map[peer.ID]*PeerData),
	}
}

// PeerData returns data associated with a given peer, if any.
func (s *Store) PeerData(pid peer.ID) (*PeerData, bool) {
	data, ok := s.peers[pid]
	return data, ok
}

// SetPeerData updates data associated with a given peer.
func (s *Store) SetPeerData(pid peer.ID, data *PeerData) {
	s.peers[pid] = data
}

// Peers returns map of peer data objects.
func (s *Store) Peers() map[peer.ID]*PeerData {
	return s.peers
}
