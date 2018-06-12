// Package p2p handles peer-to-peer networking for the sharding package.
package p2p

import (
	"github.com/ethereum/go-ethereum/log"
)

// Server is a placeholder for a shardp2p service. To be designed.
type Server struct{}

// NewServer creates a new shardp2p service instance.
func NewServer() (*Server, error) {
	return &Server{}, nil
}

// Start the main routine for an shardp2p server.
func (s *Server) Start() error {
	log.Info("Starting shardp2p server")
	return nil
}

// Stop the main shardp2p loop..
func (s *Server) Stop() error {
	log.Info("Stopping shardp2p server")
	return nil
}
