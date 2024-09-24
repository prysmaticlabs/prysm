package yamux

import (
	"context"
	"io"
	"net"
	"time"
)

type peerID string
type conn Session

type MuxedConn interface {
	io.Closer
	IsClosed() bool
	OpenStream(context.Context) (MuxedStream, error)
	AcceptStream() (MuxedStream, error)
}
type MuxedStream interface {
	io.Reader
	io.Writer
	io.Closer
	CloseWrite() error
	CloseRead() error
	Reset() error
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

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

type PeerScope interface {
	ResourceScope
	Peer() peerID
}

type Multiplexer interface {
	NewConn(c net.Conn, isServer bool, scope PeerScope) (MuxedConn, error)
}

func NewMuxedConn(m *Session) MuxedConn {
	return (*conn)(m)
}
