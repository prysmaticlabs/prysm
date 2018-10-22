package peerstore

import (
	"fmt"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
)

var _ Peerstore = (*peerstore)(nil)

type peerstore struct {
	Metrics

	KeyBook
	AddrBook
	PeerMetadata

	// lock for protocol information, separate from datastore lock
	protolock sync.Mutex
}

// NewPeerstore creates a data structure that stores peer data, backed by the
// supplied implementations of KeyBook, AddrBook and PeerMetadata.
func NewPeerstore(kb KeyBook, ab AddrBook, md PeerMetadata) Peerstore {
	return &peerstore{
		KeyBook:      kb,
		AddrBook:     ab,
		PeerMetadata: md,
		Metrics:      NewMetrics(),
	}
}

func (ps *peerstore) Peers() peer.IDSlice {
	set := map[peer.ID]struct{}{}
	for _, p := range ps.PeersWithKeys() {
		set[p] = struct{}{}
	}
	for _, p := range ps.PeersWithAddrs() {
		set[p] = struct{}{}
	}

	pps := make(peer.IDSlice, 0, len(set))
	for p := range set {
		pps = append(pps, p)
	}
	return pps
}

func (ps *peerstore) PeerInfo(p peer.ID) PeerInfo {
	return PeerInfo{
		ID:    p,
		Addrs: ps.AddrBook.Addrs(p),
	}
}

func (ps *peerstore) SetProtocols(p peer.ID, protos ...string) error {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()

	protomap := make(map[string]struct{})
	for _, proto := range protos {
		protomap[proto] = struct{}{}
	}

	return ps.Put(p, "protocols", protomap)
}

func (ps *peerstore) AddProtocols(p peer.ID, protos ...string) error {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()
	protomap, err := ps.getProtocolMap(p)
	if err != nil {
		return err
	}

	for _, proto := range protos {
		protomap[proto] = struct{}{}
	}

	return ps.Put(p, "protocols", protomap)
}

func (ps *peerstore) getProtocolMap(p peer.ID) (map[string]struct{}, error) {
	iprotomap, err := ps.Get(p, "protocols")
	switch err {
	default:
		return nil, err
	case ErrNotFound:
		return make(map[string]struct{}), nil
	case nil:
		cast, ok := iprotomap.(map[string]struct{})
		if !ok {
			return nil, fmt.Errorf("stored protocol set was not a map")
		}

		return cast, nil
	}
}

func (ps *peerstore) GetProtocols(p peer.ID) ([]string, error) {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()
	pmap, err := ps.getProtocolMap(p)
	if err != nil {
		return nil, err
	}

	var out []string
	for k, _ := range pmap {
		out = append(out, k)
	}

	return out, nil
}

func (ps *peerstore) SupportsProtocols(p peer.ID, protos ...string) ([]string, error) {
	ps.protolock.Lock()
	defer ps.protolock.Unlock()
	pmap, err := ps.getProtocolMap(p)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, proto := range protos {
		if _, ok := pmap[proto]; ok {
			out = append(out, proto)
		}
	}

	return out, nil
}

func PeerInfos(ps Peerstore, peers peer.IDSlice) []PeerInfo {
	pi := make([]PeerInfo, len(peers))
	for i, p := range peers {
		pi[i] = ps.PeerInfo(p)
	}
	return pi
}

func PeerInfoIDs(pis []PeerInfo) peer.IDSlice {
	ps := make(peer.IDSlice, len(pis))
	for i, pi := range pis {
		ps[i] = pi.ID
	}
	return ps
}
