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
	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

// Sender represents a struct that is able to relay information via p2p.
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
	log.Info("Starting service")
	if err := startDiscovery(s.ctx, s.host, s.gsub); err != nil {
		log.Errorf("Could not start p2p discovery! %v", err)
		return
	}

	// Subscribe to all topics.
	//	for topic, msgType := range topicTypeMapping {
	//		log.WithFields(logrus.Fields{
	//			"topic": topic,
	//		}).Debug("Subscribing to topic")
	//		go s.subscribeToTopic(topic, msgType)
	//	}
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	log.Info("Stopping service")

	s.cancel()
	return nil
}

// RegisterTopic, message, and the adapter stack for the given topic. The message type provided
// will be feed selector for emitting messages received on a given topic.
//
// The topics can originate from multiple sources. In other words, messages on TopicA may come
// from direct peer communication or a pub/sub channel.
//
// TODO
func (s *Server) RegisterTopic(topic string, message interface{}, adapters []Adapter) {
	var msgType reflect.Type // TODO
	log.WithFields(logrus.Fields{
		"topic": topic,
	}).Debug("Subscribing to topic")

	sub, err := s.gsub.Subscribe(topic)
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

		// TODO: Run the adapter stack.

		s.emit(feed, msg, msgType)
	}

}

// TODO: rename
func (s *Server) emit(feed *event.Feed, msg *floodsub.Message, msgType reflect.Type) {

	// TODO: reflect.Value.Interface() can panic so we should capture that
	// panic so the server doesn't crash.
	d, ok := reflect.New(msgType).Interface().(proto.Message)
	if !ok {
		log.Error("Received message is not a protobuf message")
		return
	}
	if err := proto.Unmarshal(msg.Data, d); err != nil {
		log.Errorf("Failed to decode data: %v", err)
		return
	}

	i := feed.Send(Message{Data: d})
	log.WithFields(logrus.Fields{
		"numSubs": i,
	}).Debug("Sent a request to subs")

}

// Subscribe returns a subscription to a feed of msg's Type and adds the channels to the feed.
func (s *Server) Subscribe(msg interface{}, channel interface{}) event.Subscription {
	return s.Feed(msg).Subscribe(channel)
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

	if topic == shardpb.Topic_UNKNOWN {
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
	if err := s.gsub.Publish(topic.String(), b); err != nil {
		log.Errorf("Failed to publish to gossipsub topic: %v", err)
	}
}

func (s *Server) subscribeToTopic(topic shardpb.Topic, msgType reflect.Type) {
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
		}).Debug("Sent a request to subs")
	}
}
