package peerstore

import (
	"context"
	"errors"
	"math"
	"time"

	ic "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

var ErrNotFound = errors.New("item not found")

var (
	// AddressTTL is the expiration time of addresses.
	AddressTTL = time.Hour

	// TempAddrTTL is the ttl used for a short lived address
	TempAddrTTL = time.Second * 10

	// ProviderAddrTTL is the TTL of an address we've received from a provider.
	// This is also a temporary address, but lasts longer. After this expires,
	// the records we return will require an extra lookup.
	ProviderAddrTTL = time.Minute * 10

	// RecentlyConnectedAddrTTL is used when we recently connected to a peer.
	// It means that we are reasonably certain of the peer's address.
	RecentlyConnectedAddrTTL = time.Minute * 10

	// OwnObservedAddrTTL is used for our own external addresses observed by peers.
	OwnObservedAddrTTL = time.Minute * 10
)

// Permanent TTLs (distinct so we can distinguish between them, constant as they
// are, in fact, permanent)
const (

	// PermanentAddrTTL is the ttl for a "permanent address" (e.g. bootstrap nodes).
	PermanentAddrTTL = math.MaxInt64 - iota

	// ConnectedAddrTTL is the ttl used for the addresses of a peer to whom
	// we're connected directly. This is basically permanent, as we will
	// clear them + re-add under a TempAddrTTL after disconnecting.
	ConnectedAddrTTL
)

// Peerstore provides a threadsafe store of Peer related
// information.
type Peerstore interface {
	AddrBook
	KeyBook
	PeerMetadata
	Metrics

	// PeerInfo returns a peer.PeerInfo struct for given peer.ID.
	// This is a small slice of the information Peerstore has on
	// that peer, useful to other services.
	PeerInfo(peer.ID) PeerInfo

	GetProtocols(peer.ID) ([]string, error)
	AddProtocols(peer.ID, ...string) error
	SetProtocols(peer.ID, ...string) error
	SupportsProtocols(peer.ID, ...string) ([]string, error)

	// Peers returns all of the peer IDs stored across all inner stores.
	Peers() peer.IDSlice
}

// PeerMetadata can handle values of any type. Serializing values is
// up to the implementation. Dynamic type introspection may not be
// supported, in which case explicitly enlisting types in the
// serializer may be required.
//
// Refer to the docs of the underlying implementation for more
// information.
type PeerMetadata interface {
	// Get/Put is a simple registry for other peer-related key/value pairs.
	// if we find something we use often, it should become its own set of
	// methods. this is a last resort.
	Get(p peer.ID, key string) (interface{}, error)
	Put(p peer.ID, key string, val interface{}) error
}

// AddrBook holds the multiaddrs of peers.
type AddrBook interface {

	// AddAddr calls AddAddrs(p, []ma.Multiaddr{addr}, ttl)
	AddAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration)

	// AddAddrs gives this AddrBook addresses to use, with a given ttl
	// (time-to-live), after which the address is no longer valid.
	// If the manager has a longer TTL, the operation is a no-op for that address
	AddAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration)

	// SetAddr calls mgr.SetAddrs(p, addr, ttl)
	SetAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration)

	// SetAddrs sets the ttl on addresses. This clears any TTL there previously.
	// This is used when we receive the best estimate of the validity of an address.
	SetAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration)

	// UpdateAddrs updates the addresses associated with the given peer that have
	// the given oldTTL to have the given newTTL.
	UpdateAddrs(p peer.ID, oldTTL time.Duration, newTTL time.Duration)

	// Addresses returns all known (and valid) addresses for a given peer
	Addrs(p peer.ID) []ma.Multiaddr

	// AddrStream returns a channel that gets all addresses for a given
	// peer sent on it. If new addresses are added after the call is made
	// they will be sent along through the channel as well.
	AddrStream(context.Context, peer.ID) <-chan ma.Multiaddr

	// ClearAddresses removes all previously stored addresses
	ClearAddrs(p peer.ID)

	// PeersWithAddrs returns all of the peer IDs stored in the AddrBook
	PeersWithAddrs() peer.IDSlice
}

// KeyBook tracks the keys of Peers.
type KeyBook interface {

	// PubKey stores the public key of a peer.
	PubKey(peer.ID) ic.PubKey

	// AddPubKey stores the public key of a peer.
	AddPubKey(peer.ID, ic.PubKey) error

	// PrivKey returns the private key of a peer.
	PrivKey(peer.ID) ic.PrivKey

	// AddPrivKey stores the private key of a peer.
	AddPrivKey(peer.ID, ic.PrivKey) error

	// PeersWithKeys returns all the peer IDs stored in the KeyBook
	PeersWithKeys() peer.IDSlice
}
