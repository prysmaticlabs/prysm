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
	"context"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	floodsub "github.com/libp2p/go-floodsub"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
)

// Sender represents a struct that is able to relay information via shardp2p.
// Server implements this interface.
type Sender interface {
	Send(msg interface{}, peer Peer)
}

// Server is a placeholder for a p2p service. To be designed.
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	mutex  *sync.Mutex
	feeds  map[reflect.Type]*event.Feed
	host   host.Host
	gsub   *floodsub.PubSub
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

	gsub, err := floodsub.NewGossipSub(ctx, host)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Server{
		ctx:    ctx,
		cancel: cancel,
		feeds:  make(map[reflect.Type]*event.Feed),
		host:   host,
		gsub:   gsub,
		mutex:  &sync.Mutex{},
	}, nil
}

// Start the main routine for an p2p server.
func (s *Server) Start() {
	log.Info("Starting shardp2p server")
	if err := startDiscovery(s.ctx, s.host, s.gsub); err != nil {
		log.Errorf("Could not start p2p discovery! %v", err)
		return
	}

	// Subscribe to all topics.
	for topic, msgType := range topicTypeMapping {
		log.WithFields(logrus.Fields{
			"topic": topic,
		}).Debug("Subscribing to topic")
		go s.subscribeToTopic(topic, msgType)
	}
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	log.Info("Stopping shardp2p server")

	s.cancel()
	return nil
}

// Send a message to a specific peer.
func (s *Server) Send(msg interface{}, peer Peer) {
	// TODO
	// https://github.com/prysmaticlabs/prysm/issues/175

	// TODO: Support passing value and pointer type messages.

	// TODO: Remove debug log after send is implemented.
	_ = peer
	log.Debug("Broadcasting to everyone rather than sending a single peer")
	s.Broadcast(msg)
}

// Broadcast a message to the world.
func (s *Server) Broadcast(msg interface{}) {
	// TODO https://github.com/prysmaticlabs/prysm/issues/176
	topic := topic(msg)
	log.WithFields(logrus.Fields{
		"topic": topic,
	}).Debugf("Broadcasting msg %T", msg)

	if topic == pb.Topic_UNKNOWN {
		log.Warnf("Topic is unknown for message type %T. %v", msg, msg)
	}

	// TODO: Next assertion may fail if your msg is not a pointer to a msg.
	m, ok := msg.(proto.Message)
	if !ok {
		log.Errorf("Message to broadcast (type: %T) is not a protobuf message: %v", msg, msg)
		return
	}

	b, err := proto.Marshal(m)
	if err != nil {
		log.Errorf("Failed to marshal data for broadcast: %v", err)
		return
	}
	s.gsub.Publish(topic.String(), b)
}

func (s *Server) subscribeToTopic(topic pb.Topic, msgType reflect.Type) {
	sub, err := s.gsub.Subscribe(topic.String())
	if err != nil {
		log.Errorf("Failed to subscribe to topic: %v", err)
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
			log.Errorf("Failed to get next message: %v", err)
			return
		}

		// TODO: reflect.Value.Interface() can panic so we should capture that
		// panic so the server doesn't crash.
		d, ok := reflect.New(msgType).Interface().(proto.Message)
		if !ok {
			log.Error("Received message is not a protobuf message")
			continue
		}
		err = proto.Unmarshal(msg.Data, d)
		if err != nil {
			log.Errorf("Failed to decode data: %v", err)
			continue
		}

		i := feed.Send(Message{Data: d})
		log.WithFields(logrus.Fields{
			"numSubs": i,
		}).Debug("Send a request to subs")
	}
}
