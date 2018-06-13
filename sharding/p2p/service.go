// Package p2p handles peer-to-peer networking for the sharding package.
package p2p

import (
	"reflect"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// Server is a placeholder for a shardp2p service. To be designed.
type Server struct {
	feeds map[reflect.Type]*event.Feed
}

// NewServer creates a new p2p server instance.
func NewServer() (*Server, error) {
	return &Server{
		feeds: make(map[reflect.Type]*event.Feed),
	}, nil
}

// Start the main routine for an p2p server.
func (s *Server) Start() {
	log.Info("Starting shardp2p server")
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	log.Info("Stopping shardp2p server")
	return nil
}

// Send a message to a specific peer.
func (s *Server) Send(msg interface{}, Peer peer) {
	// TODO
}

// Broadcast a message to the world.
func (s *Server) Broadcast(msg interface{}) {
	// TODO
}
