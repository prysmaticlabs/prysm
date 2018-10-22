package pstoremem

import (
	"context"
	"sort"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"

	pstore "github.com/libp2p/go-libp2p-peerstore"
	addr "github.com/libp2p/go-libp2p-peerstore/addr"
)

var log = logging.Logger("peerstore")

type expiringAddr struct {
	Addr    ma.Multiaddr
	TTL     time.Duration
	Expires time.Time
}

func (e *expiringAddr) ExpiredBy(t time.Time) bool {
	return t.After(e.Expires)
}

var _ pstore.AddrBook = (*memoryAddrBook)(nil)

// memoryAddrBook manages addresses.
type memoryAddrBook struct {
	addrmu sync.Mutex
	addrs  map[peer.ID]map[string]expiringAddr

	nextGC time.Time

	subManager *AddrSubManager
}

func NewAddrBook() pstore.AddrBook {
	return &memoryAddrBook{
		addrs:      make(map[peer.ID]map[string]expiringAddr),
		subManager: NewAddrSubManager(),
	}
}

// gc garbage collects the in-memory address book. The caller *must* hold the addrmu lock.
func (mab *memoryAddrBook) gc() {
	now := time.Now()
	if !now.After(mab.nextGC) {
		return
	}
	for p, amap := range mab.addrs {
		for k, addr := range amap {
			if addr.ExpiredBy(now) {
				delete(amap, k)
			}
		}
		if len(amap) == 0 {
			delete(mab.addrs, p)
		}
	}
	mab.nextGC = time.Now().Add(pstore.AddressTTL)
}

func (mab *memoryAddrBook) PeersWithAddrs() peer.IDSlice {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	pids := make(peer.IDSlice, 0, len(mab.addrs))
	for pid := range mab.addrs {
		pids = append(pids, pid)
	}
	return pids
}

// AddAddr calls AddAddrs(p, []ma.Multiaddr{addr}, ttl)
func (mab *memoryAddrBook) AddAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration) {
	mab.AddAddrs(p, []ma.Multiaddr{addr}, ttl)
}

// AddAddrs gives memoryAddrBook addresses to use, with a given ttl
// (time-to-live), after which the address is no longer valid.
// If the manager has a longer TTL, the operation is a no-op for that address
func (mab *memoryAddrBook) AddAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration) {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	// if ttl is zero, exit. nothing to do.
	if ttl <= 0 {
		return
	}

	amap := mab.addrs[p]
	if amap == nil {
		amap = make(map[string]expiringAddr, len(addrs))
		mab.addrs[p] = amap
	}
	exp := time.Now().Add(ttl)
	for _, addr := range addrs {
		if addr == nil {
			log.Warningf("was passed nil multiaddr for %s", p)
			continue
		}
		addrstr := string(addr.Bytes())
		a, found := amap[addrstr]
		if !found || exp.After(a.Expires) {
			amap[addrstr] = expiringAddr{Addr: addr, Expires: exp, TTL: ttl}

			mab.subManager.BroadcastAddr(p, addr)
		}
	}
	mab.gc()
}

// SetAddr calls mgr.SetAddrs(p, addr, ttl)
func (mab *memoryAddrBook) SetAddr(p peer.ID, addr ma.Multiaddr, ttl time.Duration) {
	mab.SetAddrs(p, []ma.Multiaddr{addr}, ttl)
}

// SetAddrs sets the ttl on addresses. This clears any TTL there previously.
// This is used when we receive the best estimate of the validity of an address.
func (mab *memoryAddrBook) SetAddrs(p peer.ID, addrs []ma.Multiaddr, ttl time.Duration) {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	amap := mab.addrs[p]
	if amap == nil {
		amap = make(map[string]expiringAddr, len(addrs))
		mab.addrs[p] = amap
	}

	exp := time.Now().Add(ttl)
	for _, addr := range addrs {
		if addr == nil {
			log.Warningf("was passed nil multiaddr for %s", p)
			continue
		}
		// re-set all of them for new ttl.
		addrstr := string(addr.Bytes())

		if ttl > 0 {
			amap[addrstr] = expiringAddr{Addr: addr, Expires: exp, TTL: ttl}

			mab.subManager.BroadcastAddr(p, addr)
		} else {
			delete(amap, addrstr)
		}
	}
	mab.gc()
}

// UpdateAddrs updates the addresses associated with the given peer that have
// the given oldTTL to have the given newTTL.
func (mab *memoryAddrBook) UpdateAddrs(p peer.ID, oldTTL time.Duration, newTTL time.Duration) {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	amap, found := mab.addrs[p]
	if !found {
		return
	}

	exp := time.Now().Add(newTTL)
	for k, addr := range amap {
		if oldTTL == addr.TTL {
			addr.TTL = newTTL
			addr.Expires = exp
			amap[k] = addr
		}
	}
	mab.gc()
}

// Addresses returns all known (and valid) addresses for a given
func (mab *memoryAddrBook) Addrs(p peer.ID) []ma.Multiaddr {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	amap, found := mab.addrs[p]
	if !found {
		return nil
	}

	now := time.Now()
	good := make([]ma.Multiaddr, 0, len(amap))
	for k, m := range amap {
		if !m.ExpiredBy(now) {
			good = append(good, m.Addr)
		} else {
			delete(amap, k)
		}
	}

	return good
}

// ClearAddrs removes all previously stored addresses
func (mab *memoryAddrBook) ClearAddrs(p peer.ID) {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	delete(mab.addrs, p)
}

// AddrStream returns a channel on which all new addresses discovered for a
// given peer ID will be published.
func (mab *memoryAddrBook) AddrStream(ctx context.Context, p peer.ID) <-chan ma.Multiaddr {
	mab.addrmu.Lock()
	defer mab.addrmu.Unlock()

	baseaddrslice := mab.addrs[p]
	initial := make([]ma.Multiaddr, 0, len(baseaddrslice))
	for _, a := range baseaddrslice {
		initial = append(initial, a.Addr)
	}

	return mab.subManager.AddrStream(ctx, p, initial)
}

type addrSub struct {
	pubch  chan ma.Multiaddr
	lk     sync.Mutex
	buffer []ma.Multiaddr
	ctx    context.Context
}

func (s *addrSub) pubAddr(a ma.Multiaddr) {
	select {
	case s.pubch <- a:
	case <-s.ctx.Done():
	}
}

// An abstracted, pub-sub manager for address streams. Extracted from
// memoryAddrBook in order to support additional implementations.
type AddrSubManager struct {
	mu   sync.RWMutex
	subs map[peer.ID][]*addrSub
}

// NewAddrSubManager initializes an AddrSubManager.
func NewAddrSubManager() *AddrSubManager {
	return &AddrSubManager{
		subs: make(map[peer.ID][]*addrSub),
	}
}

// Used internally by the address stream coroutine to remove a subscription
// from the manager.
func (mgr *AddrSubManager) removeSub(p peer.ID, s *addrSub) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	subs := mgr.subs[p]
	if len(subs) == 1 {
		if subs[0] != s {
			return
		}
		delete(mgr.subs, p)
		return
	}

	for i, v := range subs {
		if v == s {
			subs[i] = subs[len(subs)-1]
			subs[len(subs)-1] = nil
			mgr.subs[p] = subs[:len(subs)-1]
			return
		}
	}
}

// BroadcastAddr broadcasts a new address to all subscribed streams.
func (mgr *AddrSubManager) BroadcastAddr(p peer.ID, addr ma.Multiaddr) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if subs, ok := mgr.subs[p]; ok {
		for _, sub := range subs {
			sub.pubAddr(addr)
		}
	}
}

// AddrStream creates a new subscription for a given peer ID, pre-populating the
// channel with any addresses we might already have on file.
func (mgr *AddrSubManager) AddrStream(ctx context.Context, p peer.ID, initial []ma.Multiaddr) <-chan ma.Multiaddr {
	sub := &addrSub{pubch: make(chan ma.Multiaddr), ctx: ctx}
	out := make(chan ma.Multiaddr)

	mgr.mu.Lock()
	if _, ok := mgr.subs[p]; ok {
		mgr.subs[p] = append(mgr.subs[p], sub)
	} else {
		mgr.subs[p] = []*addrSub{sub}
	}
	mgr.mu.Unlock()

	sort.Sort(addr.AddrList(initial))

	go func(buffer []ma.Multiaddr) {
		defer close(out)

		sent := make(map[string]bool, len(buffer))
		var outch chan ma.Multiaddr

		for _, a := range buffer {
			sent[string(a.Bytes())] = true
		}

		var next ma.Multiaddr
		if len(buffer) > 0 {
			next = buffer[0]
			buffer = buffer[1:]
			outch = out
		}

		for {
			select {
			case outch <- next:
				if len(buffer) > 0 {
					next = buffer[0]
					buffer = buffer[1:]
				} else {
					outch = nil
					next = nil
				}
			case naddr := <-sub.pubch:
				if sent[string(naddr.Bytes())] {
					continue
				}

				sent[string(naddr.Bytes())] = true
				if next == nil {
					next = naddr
					outch = out
				} else {
					buffer = append(buffer, naddr)
				}
			case <-ctx.Done():
				mgr.removeSub(p, sub)
				return
			}
		}

	}(initial)

	return out
}
