// Package p2p handles peer-to-peer networking for the sharding package.
package p2p

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/p2p/protocol"
	"github.com/ethereum/go-ethereum/sharding/p2p/topics"

	floodsub "github.com/libp2p/go-floodsub"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	ps "github.com/libp2p/go-libp2p-peerstore"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery"
)

var logger = log.New()

// Server is a placeholder for a p2p service. To be designed.
type Server struct {
	ctx         context.Context
	cancel      context.CancelFunc
	feeds       map[reflect.Type]*event.Feed
	host        host.Host
	protocols   []protocol.Protocol
	gsub        *floodsub.PubSub
	mdnsService mdns.Service
}

// NewServer creates a new p2p server instance.
func NewServer() (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := buildOptions()
	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		cancel()
		return nil, err
	}

	// TODO: Is this the best place for this?
	gsub, err := floodsub.NewGossipSub(ctx, host) // TODO: Add opts

	// TODO: handle protocol requests to feeds and from send/broadcast.
	protocols := []protocol.Protocol{
		protocol.NewPingProtocol(host),
	}

	// TODO: Is this the best place for this?
	mdnsService, err := mdns.NewMdnsService(ctx, host, 15*time.Second, "")
	if err != nil {
		cancel()
		return nil, err
	}

	mdnsService.RegisterNotifee(&thing{host, gsub, ctx})

	return &Server{
		ctx:         ctx,
		cancel:      cancel,
		feeds:       make(map[reflect.Type]*event.Feed),
		host:        host,
		protocols:   protocols,
		gsub:        gsub,
		mdnsService: mdnsService,
	}, nil
}

// Start the main routine for an p2p server.
func (s *Server) Start() {
	logger.Info("Starting shardp2p server")

	go func() {
		sub, err := s.gsub.Subscribe(topics.Ping)
		if err != nil {
			logger.Crit(fmt.Sprintf("Failed to sub to ping: %v", err))
		}
		defer sub.Cancel()

		for {
			msg, err := sub.Next(s.ctx)
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				log.Error(fmt.Sprintf("Failed to get next message: %v", err))
				return
			} else {
				log.Info(fmt.Sprintf("Received raw message: %s", msg.Data))

			}
		}
	}()
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

	// DEBUG: Ping everyone constantly
	logger.Info("broadcasting a ping")
	s.gsub.Publish(topics.Ping, []byte("Ping!"))
	// 	p := s.protocols[0].(*protocol.PingProtocol)
	// 	if p == nil {
	// 		logger.Info("p is nil")
	// 	}
	// 	p.Ping(s.ctx)
}

type thing struct {
	host host.Host
	gsub *floodsub.PubSub
	ctx  context.Context
}

func (t *thing) HandlePeerFound(pi ps.PeerInfo) {
	logger.Info(fmt.Sprintf("Peer found. What do we do with it? %s", pi))

	t.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	if err := t.host.Connect(t.ctx, pi); err != nil {
		logger.Info(fmt.Sprintf("Failed to connect to peer: %v", err))
	}

	logger.Info(fmt.Sprintf("Peers now: %s", t.host.Peerstore().Peers()))
	logger.Info(fmt.Sprintf("gsub has peers: %s", t.gsub.ListPeers(topics.Ping)))
}
