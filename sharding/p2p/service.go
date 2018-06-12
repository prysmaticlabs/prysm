// Package p2p handles peer-to-peer networking for the sharding package.
package p2p

import (
	"reflect"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// PeerInfo defines an interface to extract important information about a shardp2p
// peer.
type PeerInfo interface{}

// Message defines a shardp2p message received via a subscription.
type Message struct {
	Peer PeerInfo
	Data interface{} // Of the message type requested from p2p.Server.Feed(...)
}

// Server is a placeholder for a shardp2p service. To be designed.
type Server struct {
	feeds map[reflect.Type]*event.Feed
}

// NewServer creates a new shardp2p service instance.
func NewServer() (*Server, error) {
	return &Server{
		feeds: make(map[reflect.Type]*event.Feed),
	}, nil
}

// Start the main routine for an shardp2p server.
func (s *Server) Start() error {
	log.Info("Starting shardp2p server")
	return nil
}

// Stop the main shardp2p loop...
func (s *Server) Stop() error {
	log.Info("Stopping shardp2p server")
	return nil
}

func (s *Server) Feed(msg interface{}) (*event.Feed, error) {
	t := reflect.TypeOf(msg)
	if s.feeds[t] == nil {
		s.feeds[t] = new(event.Feed)
	}
	return s.feeds[t], nil
}

func (s *Server) Broadcast(message interface{}) error {
	return nil
}

func (s *Server) Send(message interface{}, peer PeerInfo) error {
	return nil
}
