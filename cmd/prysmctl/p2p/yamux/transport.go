package yamux

import (
	"net"
)

const ID = "/yamux/1.0.0"

type Transport Config

var DefaultTransport *Transport

func init() {
	config := DefaultConfig()
	DefaultTransport = (*Transport)(config)
}

var _ Multiplexer = &Transport{}

func (t *Transport) NewConn(nc net.Conn, isServer bool, scope PeerScope) (*Session, error) {
	var newSpan func() (MemoryManager, error)
	if scope != nil {
		newSpan = func() (MemoryManager, error) { return scope.BeginSpan() }
	}

	var s *Session
	var err error
	if isServer {
		s, err = Server(nc, t.Config(), newSpan)
	} else {
		s, err = Client(nc, t.Config(), newSpan)
	}
	if err != nil {
		return nil, err
	}
	return NewMuxedConn(s), nil
}

func (t *Transport) Config() *Config {
	return (*Config)(t)
}
