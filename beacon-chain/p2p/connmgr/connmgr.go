/*
Copyright 2019. Protocol Labs, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Package connmgr : This file is forked from github.com/libp2p/go-libp2p-core/connmgr/connmgr.go
package connmgr

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/runutil"
)

// SilencePeriod refers to the period in which a connection is given leeway by the connection
// manager before it is handled normally.
var SilencePeriod = 5 * time.Second

// TickerPeriod represents the frequency in which we check the number of
// open connections.
var TickerPeriod = 5 * time.Second

var log = logging.Logger("connmgr")

// BasicConnMgr is a ConnManager that trims connections whenever the count exceeds the
// high watermark. New connections are given a grace period before they're subject
// to trimming. Trims are automatically run on demand, only if the time from the
// previous trim is higher than 10 seconds. Furthermore, trims can be explicitly
// requested through the public interface of this struct (see TrimOpenConns).
//
// See configuration parameters in NewConnManager.
type BasicConnMgr struct {
	highWater   int
	lowWater    int
	connCount   int32
	gracePeriod time.Duration
	segments    segments

	plk       sync.RWMutex
	protected map[peer.ID]map[string]struct{}

	// channel-based semaphore that enforces only a single trim is in progress
	trimRunningCh chan struct{}
	lastTrim      time.Time
	silencePeriod time.Duration

	ctx    context.Context
	cancel func()
}

var _ connmgr.ConnManager = (*BasicConnMgr)(nil)

type segment struct {
	sync.Mutex
	peers map[peer.ID]*peerInfo
}

type segments [256]*segment

func (ss *segments) get(p peer.ID) *segment {
	return ss[p[len(p)-1]]
}

func (ss *segments) countPeers() (count int) {
	for _, seg := range ss {
		seg.Lock()
		count += len(seg.peers)
		seg.Unlock()
	}
	return count
}

func (s *segment) tagInfoFor(p peer.ID) *peerInfo {
	pi, ok := s.peers[p]
	if ok {
		return pi
	}
	// create a temporary peer to buffer early tags before the Connected notification arrives.
	pi = &peerInfo{
		id:        p,
		firstSeen: time.Now(), // this timestamp will be updated when the first Connected notification arrives.
		temp:      true,
		tags:      make(map[string]int),
		conns:     make(map[network.Conn]time.Time),
	}
	s.peers[p] = pi
	return pi
}

// NewConnManager creates a new BasicConnMgr with the provided params:
// * lo and hi are watermarks governing the number of connections that'll be maintained.
//   When the peer count exceeds the 'high watermark', as many peers will be pruned (and
//   their connections terminated) until 'low watermark' peers remain.
// * grace is the amount of time a newly opened connection is given before it becomes
//   subject to pruning.
func NewConnManager(low, hi int, grace time.Duration) *BasicConnMgr {
	ctx, cancel := context.WithCancel(context.Background())
	cm := &BasicConnMgr{
		highWater:     hi,
		lowWater:      low,
		gracePeriod:   grace,
		trimRunningCh: make(chan struct{}, 1),
		protected:     make(map[peer.ID]map[string]struct{}, 16),
		silencePeriod: SilencePeriod,
		ctx:           ctx,
		cancel:        cancel,
		segments: func() (ret segments) {
			for i := range ret {
				ret[i] = &segment{
					peers: make(map[peer.ID]*peerInfo),
				}
			}
			return ret
		}(),
	}

	// Check every TickerPeriod to see if we should trim the number of active connections.
	runutil.RunEvery(cm.ctx, TickerPeriod, func() {
		if atomic.LoadInt32(&cm.connCount) > int32(cm.highWater) {
			cm.TrimOpenConns(cm.ctx)
		}
	})

	return cm
}

// Close shutsdown the connection manager.
func (cm *BasicConnMgr) Close() error {
	cm.cancel()
	return nil
}

// Protect is used to protect a peer from being pruned by the
// connection manager.
func (cm *BasicConnMgr) Protect(id peer.ID, tag string) {
	cm.plk.Lock()
	defer cm.plk.Unlock()

	tags, ok := cm.protected[id]
	if !ok {
		tags = make(map[string]struct{}, 2)
		cm.protected[id] = tags
	}
	tags[tag] = struct{}{}
}

// Unprotect is used to remove the protection a previously protected peer
// so that it can be normally pruned by the connection manager.
func (cm *BasicConnMgr) Unprotect(id peer.ID, tag string) (protected bool) {
	cm.plk.Lock()
	defer cm.plk.Unlock()

	tags, ok := cm.protected[id]
	if !ok {
		return false
	}
	if delete(tags, tag); len(tags) == 0 {
		delete(cm.protected, id)
		return false
	}
	return true
}

// peerInfo stores metadata for a given peer.
type peerInfo struct {
	id    peer.ID
	tags  map[string]int // value for each tag
	value int            // cached sum of all tag values
	temp  bool           // this is a temporary entry holding early tags, and awaiting connections

	conns map[network.Conn]time.Time // start time of each connection

	firstSeen time.Time // timestamp when we began tracking this peer.
}

// TrimOpenConns closes the connections of as many peers as needed to make the peer count
// equal the low watermark. Peers are sorted in ascending order based on their total value,
// pruning those peers with the lowest scores first, as long as they are not within their
// grace period.
//
// (a) there's another trim in progress, or (b) the silence period is in effect.
func (cm *BasicConnMgr) TrimOpenConns(ctx context.Context) {
	select {
	case cm.trimRunningCh <- struct{}{}:
	default:
		return
	}
	defer func() { <-cm.trimRunningCh }()
	if time.Since(cm.lastTrim) < cm.silencePeriod {
		// skip this attempt to trim as the last one just took place.
		return
	}

	defer log.EventBegin(ctx, "connCleanup").Done()
	for _, c := range cm.getConnsToClose(ctx) {
		log.Info("closing conn: ", c.RemotePeer())
		log.Event(ctx, "closeConn", c.RemotePeer())
		c.Close()
	}

	cm.lastTrim = time.Now()
}

// getConnsToClose runs the heuristics described in TrimOpenConns and returns the
// connections to close.
func (cm *BasicConnMgr) getConnsToClose(ctx context.Context) []network.Conn {
	if cm.lowWater == 0 || cm.highWater == 0 {
		// disabled
		return nil
	}

	nconns := int(atomic.LoadInt32(&cm.connCount))
	if nconns <= cm.lowWater {
		log.Info("open connection count below limit")
		return nil
	}

	npeers := cm.segments.countPeers()
	candidates := make([]*peerInfo, 0, npeers)
	ncandidates := 0
	gracePeriodStart := time.Now().Add(-cm.gracePeriod)

	cm.plk.RLock()
	for _, s := range cm.segments {
		s.Lock()
		for id, inf := range s.peers {
			if _, ok := cm.protected[id]; ok {
				// skip over protected peer.
				continue
			}
			if inf.firstSeen.After(gracePeriodStart) {
				// skip peers in the grace period.
				continue
			}
			candidates = append(candidates, inf)
			ncandidates += len(inf.conns)
		}
		s.Unlock()
	}
	cm.plk.RUnlock()

	if ncandidates < cm.lowWater {
		log.Info("open connection count above limit but too many are in the grace period")
		// We have too many connections but fewer than lowWater
		// connections out of the grace period.
		//
		// If we trimmed now, we'd kill potentially useful connections.
		return nil
	}

	// Sort peers according to their value.
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		// temporary peers are preferred for pruning.
		if left.temp != right.temp {
			return left.temp
		}
		// otherwise, compare by value.
		return left.value < right.value
	})

	target := ncandidates - cm.lowWater

	// slightly overallocate because we may have more than one conns per peer
	selected := make([]network.Conn, 0, target+10)

	for _, inf := range candidates {
		if target <= 0 {
			break
		}

		// lock this to protect from concurrent modifications from connect/disconnect events
		s := cm.segments.get(inf.id)
		s.Lock()

		if len(inf.conns) == 0 && inf.temp {
			// handle temporary entries for early tags -- this entry has gone past the grace period
			// and still holds no connections, so prune it.
			delete(s.peers, inf.id)
		} else {
			for c := range inf.conns {
				selected = append(selected, c)
			}
		}
		target -= len(inf.conns)
		s.Unlock()
	}

	return selected
}

// GetTagInfo is called to fetch the tag information associated with a given
// peer, nil is returned if p refers to an unknown peer.
func (cm *BasicConnMgr) GetTagInfo(p peer.ID) *connmgr.TagInfo {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi, ok := s.peers[p]
	if !ok {
		return nil
	}

	out := &connmgr.TagInfo{
		FirstSeen: pi.firstSeen,
		Value:     pi.value,
		Tags:      make(map[string]int),
		Conns:     make(map[string]time.Time),
	}

	for t, v := range pi.tags {
		out.Tags[t] = v
	}
	for c, t := range pi.conns {
		out.Conns[c.RemoteMultiaddr().String()] = t
	}

	return out
}

// TagPeer is called to associate a string and integer with a given peer.
func (cm *BasicConnMgr) TagPeer(p peer.ID, tag string, val int) {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi := s.tagInfoFor(p)

	// Update the total value of the peer.
	pi.value += val - pi.tags[tag]
	pi.tags[tag] = val
}

// UntagPeer is called to disassociate a string and integer from a given peer.
func (cm *BasicConnMgr) UntagPeer(p peer.ID, tag string) {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi, ok := s.peers[p]
	if !ok {
		log.Info("tried to remove tag from untracked peer: ", p)
		return
	}

	// Update the total value of the peer.
	pi.value -= pi.tags[tag]
	delete(pi.tags, tag)
}

// UpsertTag is called to insert/update a peer tag
func (cm *BasicConnMgr) UpsertTag(p peer.ID, tag string, upsert func(int) int) {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi := s.tagInfoFor(p)

	oldval := pi.tags[tag]
	newval := upsert(oldval)
	pi.value += newval - oldval
	pi.tags[tag] = newval
}

// CMInfo holds the configuration for BasicConnMgr, as well as status data.
type CMInfo struct {
	// The low watermark, as described in NewConnManager.
	LowWater int

	// The high watermark, as described in NewConnManager.
	HighWater int

	// The timestamp when the last trim was triggered.
	LastTrim time.Time

	// The configured grace period, as described in NewConnManager.
	GracePeriod time.Duration

	// The current connection count.
	ConnCount int
}

// GetInfo returns the configuration and status data for this connection manager.
func (cm *BasicConnMgr) GetInfo() CMInfo {
	return CMInfo{
		HighWater:   cm.highWater,
		LowWater:    cm.lowWater,
		LastTrim:    cm.lastTrim,
		GracePeriod: cm.gracePeriod,
		ConnCount:   int(atomic.LoadInt32(&cm.connCount)),
	}
}

// Notifee returns a sink through which Notifiers can inform the BasicConnMgr when
// events occur. Currently, the notifee only reacts upon connection events
// {Connected, Disconnected}.
func (cm *BasicConnMgr) Notifee() network.Notifiee {
	return (*cmNotifee)(cm)
}

type cmNotifee BasicConnMgr

func (nn *cmNotifee) cm() *BasicConnMgr {
	return (*BasicConnMgr)(nn)
}

// Connected is called by notifiers to inform that a new connection has been established.
// The notifee updates the BasicConnMgr to start tracking the connection. If the new connection
// count exceeds the high watermark, a trim may be triggered.
func (nn *cmNotifee) Connected(n network.Network, c network.Conn) {
	cm := nn.cm()

	p := c.RemotePeer()
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	id := c.RemotePeer()
	pinfo, ok := s.peers[id]
	if !ok {
		pinfo = &peerInfo{
			id:        id,
			firstSeen: time.Now(),
			tags:      make(map[string]int),
			conns:     make(map[network.Conn]time.Time),
		}
		s.peers[id] = pinfo
	} else if pinfo.temp {
		// we had created a temporary entry for this peer to buffer early tags before the
		// Connected notification arrived: flip the temporary flag, and update the firstSeen
		// timestamp to the real one.
		pinfo.temp = false
		pinfo.firstSeen = time.Now()
	}

	_, ok = pinfo.conns[c]
	if ok {
		log.Error("received connected notification for conn we are already tracking: ", p)
		return
	}

	pinfo.conns[c] = time.Now()
	atomic.AddInt32(&cm.connCount, 1)
}

// Disconnected is called by notifiers to inform that an existing connection has been closed or terminated.
// The notifee updates the BasicConnMgr accordingly to stop tracking the connection, and performs housekeeping.
func (nn *cmNotifee) Disconnected(n network.Network, c network.Conn) {
	cm := nn.cm()

	p := c.RemotePeer()
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	cinf, ok := s.peers[p]
	if !ok {
		log.Error("received disconnected notification for peer we are not tracking: ", p)
		return
	}

	_, ok = cinf.conns[c]
	if !ok {
		log.Error("received disconnected notification for conn we are not tracking: ", p)
		return
	}

	delete(cinf.conns, c)
	if len(cinf.conns) == 0 {
		delete(s.peers, p)
	}
	atomic.AddInt32(&cm.connCount, -1)
}

// Listen is no-op in this implementation.
func (nn *cmNotifee) Listen(n network.Network, addr ma.Multiaddr) {}

// ListenClose is no-op in this implementation.
func (nn *cmNotifee) ListenClose(n network.Network, addr ma.Multiaddr) {}

// OpenedStream is no-op in this implementation.
func (nn *cmNotifee) OpenedStream(network.Network, network.Stream) {}

// ClosedStream is no-op in this implementation.
func (nn *cmNotifee) ClosedStream(network.Network, network.Stream) {}
