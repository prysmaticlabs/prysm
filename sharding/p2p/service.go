// Package p2p handles peer-to-peer networking for the sharding package.
//
// Notes:
// Gossip sub topics can be identified by their proto message types.
//
// 		topic := proto.MessageName(myMsg)
//
// Then we can assume that only these message types are broadcast in that
// gossip subscription.
package p2p

import (
	"bytes"
	"context"
	"encoding/gob"
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
	mdnsService, err := mdns.NewMdnsService(ctx, host, 60*time.Second, "")
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

	// Subscribe to all topics.
	for topic, msgType := range topicTypeMapping {
		logger.Info(fmt.Sprintf("Registering topic: %s", topic))
		go func() {
			sub, err := s.gsub.Subscribe(topic.String())
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to subscribe to topic: %v", err))
				return
			}
			defer sub.Cancel()
			feed := s.Feed(msgType)

			for {
				msg, err := sub.Next(s.ctx)
				if s.ctx.Err() != nil {
					return // Context closed or something.
				}
				if err != nil {
					logger.Error(fmt.Sprintf("Failed to get next message: %v", err))
					return
				}
				data := reflect.New(msgType)
				buf := bytes.NewBuffer(msg.Data)
				dec := gob.NewDecoder(buf)
				err = dec.Decode(&data)
				if err != nil {
					logger.Error(fmt.Sprintf("Failed to decode data: %v", err))
					continue
				}

				i := feed.Send(Message{Data: data})
				logger.Info(fmt.Sprintf("Send a request to %d subs", i))
			}
		}()
	}
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

	// DEBUG
	logger.Warn("Debug: Broadcasting to everyone rather than sending a single peer")
	s.Broadcast(msg)
}

// Broadcast a message to the world.
func (s *Server) Broadcast(msg interface{}) {
	// TODO
	// https://github.com/prysmaticlabs/geth-sharding/issues/176

	logger.Info("broadcasting msg")
	topic := typeTopicMapping[reflect.TypeOf(msg)]
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(msg)
	if err != nil {
		log.Error(fmt.Sprintf("Error encoding error: %v", err))
		return
	}
	s.gsub.Publish(topic.String(), buf.Bytes())
}

type thing struct {
	host host.Host
	gsub *floodsub.PubSub
	ctx  context.Context
}

func (t *thing) HandlePeerFound(pi ps.PeerInfo) {
	t.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	if err := t.host.Connect(t.ctx, pi); err != nil {
		logger.Info(fmt.Sprintf("Failed to connect to peer: %v", err))
	}

	logger.Info(fmt.Sprintf("Peers now: %s", t.host.Peerstore().Peers()))
	logger.Info(fmt.Sprintf("gsub has peers: %s", t.gsub.ListPeers(topics.Ping)))
}
