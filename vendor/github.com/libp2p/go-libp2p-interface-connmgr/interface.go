package ifconnmgr

import (
	"context"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
)

// ConnManager tracks connections to peers, and allows consumers to associate metadata
// with each peer.
//
// It enables connections to be trimmed based on implementation-defined heuristics.
type ConnManager interface {

	// TagPeer tags a peer with a string, associating a weight with the tag.
	TagPeer(peer.ID, string, int)

	// Untag removes the tagged value from the peer.
	UntagPeer(p peer.ID, tag string)

	// GetTagInfo returns the metadata associated with the peer,
	// or nil if no metadata has been recorded for the peer.
	GetTagInfo(p peer.ID) *TagInfo

	// TrimOpenConns terminates open connections based on an implementation-defined
	// heuristic.
	TrimOpenConns(ctx context.Context)

	// Notifee returns an implementation that can be called back to inform of
	// opened and closed connections.
	Notifee() inet.Notifiee
}

// TagInfo stores metadata associated with a peer.
type TagInfo struct {
	FirstSeen time.Time
	Value     int

	// Tags maps tag ids to the numerical values.
	Tags map[string]int

	// Conns maps connection ids (such as remote multiaddr) to their creation time.
	Conns map[string]time.Time
}
