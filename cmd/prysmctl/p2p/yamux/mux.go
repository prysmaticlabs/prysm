package yamux

import (
	"net"
)

type ResourceScopeSpan interface {
	ResourceScope
	Done()
}

type ResourceScope interface {
	ReserveMemory(size int, prio uint8) error
	ReleaseMemory(size int)
	// Stat() ScopeStat
	BeginSpan() (ResourceScopeSpan, error)
}

// Memory manager
type PeerScope interface {
	ResourceScope
	Peer() string
}

type Multiplexer interface {
	NewConn(c net.Conn, isServer bool, scope PeerScope) (*Session, error)
}

func NewMuxedConn(m *Session) *Session {
	return m
}
