package p2p

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/sirupsen/logrus"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
)

// Sender represents a struct that is able to relay information via p2p.
// Server implements this interface.
type Sender interface {
	Send(msg interface{}, peer Peer)
}

// Server is a placeholder for a p2p service. To be designed.
type Server struct {
	ctx          context.Context
	cancel       context.CancelFunc
	mutex        *sync.Mutex
	feeds        map[reflect.Type]Feed
	host         host.Host
	gsub         *pubsub.PubSub
	topicMapping map[reflect.Type]string
}

// NewServer creates a new p2p server instance.
func NewServer() (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := buildOptions()
	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		cancel()
		return nil, err
	}

	dht := kaddht.NewDHT(ctx, h, dsync.MutexWrap(ds.NewMapDatastore()))
	// Wrap host with a routed host so that peers can be looked up in the
	// distributed hash table by their peer ID.
	h = rhost.Wrap(h, dht)

	gsub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Server{
		ctx:          ctx,
		cancel:       cancel,
		feeds:        make(map[reflect.Type]Feed),
		host:         h,
		gsub:         gsub,
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}, nil
}

// Start the main routine for an p2p server.
func (s *Server) Start() {
	log.Info("Starting service")
	if err := startDiscovery(s.ctx, s.host); err != nil {
		log.Errorf("Could not start p2p discovery! %v", err)
		return
	}
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	log.Info("Stopping service")

	s.cancel()
	return nil
}

// RegisterTopic with a message and the adapter stack for the given topic. The
// message type provided will be feed selector for emitting messages received
// on a given topic.
//
// The topics can originate from multiple sources. In other words, messages on
// TopicA may come from direct peer communication or a pub/sub channel.
func (s *Server) RegisterTopic(topic string, message proto.Message, adapters ...Adapter) {
	log.WithFields(logrus.Fields{
		"topic": topic,
	}).Debug("Subscribing to topic")

	msgType := messageType(message)
	s.topicMapping[msgType] = topic

	sub, err := s.gsub.Subscribe(topic)
	if err != nil {
		log.Errorf("Failed to subscribe to topic: %v", err)
		return
	}
	feed := s.Feed(message)

	// Reverse adapter order
	for i := len(adapters)/2 - 1; i >= 0; i-- {
		opp := len(adapters) - 1 - i
		adapters[i], adapters[opp] = adapters[opp], adapters[i]
	}

	go func() {
		defer sub.Cancel()
		for {
			msg, err := sub.Next(s.ctx)

			if s.ctx.Err() != nil {
				log.WithError(s.ctx.Err()).Debug("Context error")
				return
			}

			if err != nil {
				log.Errorf("Failed to get next message: %v", err)
				return
			}

			var h Handler = func(pMsg Message) {
				s.emit(pMsg, feed, msg, msgType)
			}

			pMsg := Message{Ctx: s.ctx}

			for _, adapter := range adapters {
				h = adapter(h)
			}

			h(pMsg)
		}
	}()
}

func (s *Server) emit(pMsg Message, feed Feed, msg *pubsub.Message, msgType reflect.Type) {
	d, ok := reflect.New(msgType).Interface().(proto.Message)
	if !ok {
		log.Errorf("Received message is not a protobuf message: %s", msgType)
		return
	}

	if err := proto.Unmarshal(msg.Data, d); err != nil {
		log.Errorf("Failed to decode data: %v", err)
		return
	}

	pMsg.Data = d

	i := feed.Send(pMsg)
	log.WithFields(logrus.Fields{
		"numSubs": i,
		"msgType": fmt.Sprintf("%T", d),
		"msgName": proto.MessageName(d),
	}).Debug("Emit p2p message to feed subscribers")
}

// Subscribe returns a subscription to a feed of msg's Type and adds the channels to the feed.
func (s *Server) Subscribe(msg proto.Message, channel chan Message) event.Subscription {
	return s.Feed(msg).Subscribe(channel)
}

// Send a message to a specific peer.
func (s *Server) Send(msg proto.Message, peer Peer) {
	// TODO(#175)
	// https://github.com/prysmaticlabs/prysm/issues/175

	// TODO(#175): Remove debug log after send is implemented.
	_ = peer
	log.Debug("Broadcasting to everyone rather than sending a single peer")
	s.Broadcast(msg)
}

// Broadcast a message to the world.
func (s *Server) Broadcast(msg proto.Message) {
	topic := s.topicMapping[messageType(msg)]
	log.WithFields(logrus.Fields{
		"topic": topic,
	}).Debugf("Broadcasting msg %+v", msg)

	if topic == "" {
		log.Warnf("Topic is unknown for message type %T. %v", msg, msg)
	}

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
	if err := s.gsub.Publish(topic, b); err != nil {
		log.Errorf("Failed to publish to gossipsub topic: %v", err)
	}
}
