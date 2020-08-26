package peers

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// peerDataStore is a container for various peer related data (both protocol and app level).
// Container implements RWMutex, so data access can be restricted on the container level. This allows
// different components rely on the very same peer map container.
type peerDataStore struct {
	sync.RWMutex
	ctx    context.Context
	config *peerDataStoreConfig
	peers  map[peer.ID]*peerData
}

// peerDataStoreConfig holds peer store parameters.
type peerDataStoreConfig struct {
	maxPeers int
}

// peerData aggregates protocol and application level info about a single peer.
type peerData struct {
	address               ma.Multiaddr
	direction             network.Direction
	connState             PeerConnectionState
	chainState            *pb.Status
	enr                   *enr.Record
	metaData              *pb.MetaData
	chainStateLastUpdated time.Time
	badResponses          int
	processedBlocks       uint64
	blockProviderUpdated  time.Time
}

// newPeerDataStore creates peer store.
func newPeerDataStore(ctx context.Context, config *peerDataStoreConfig) *peerDataStore {
	return &peerDataStore{
		ctx:    ctx,
		config: config,
		peers:  make(map[peer.ID]*peerData),
	}
}
