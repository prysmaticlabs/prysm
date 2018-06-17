// Package p2p handles peer-to-peer networking for the sharding package.
package p2p

import (
	"context"
	"reflect"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
)

var logger = log.New()

// Server is a placeholder for a p2p service. To be designed.
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	feeds  map[reflect.Type]*event.Feed
	host   host.Host
}

// NewServer creates a new p2p server instance.
func NewServer() (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := buildOptions()
	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Server{
		ctx:    ctx,
		cancel: cancel,
		feeds:  make(map[reflect.Type]*event.Feed),
		host:   host,
	}, nil
}

// Start the main routine for an p2p server.
func (s *Server) Start() {
	logger.Info("Starting shardp2p server")
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	logger.Info("Stopping shardp2p server")
	s.cancel()
	return nil
}

// Send a message to a specific peer.
func (s *Server) Send(msg interface{}, peer Peer) {
	// TODO
	// https://github.com/prysmaticlabs/geth-sharding/issues/175
}

// Broadcast a message to the world.
func (s *Server) Broadcast(msg interface{}) {
	// TODO
	// https://github.com/prysmaticlabs/geth-sharding/issues/176
}
