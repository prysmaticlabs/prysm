package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/gogo/protobuf/proto"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Sender represents a struct that is able to relay information via p2p.
// Server implements this interface.
type Sender interface {
	Send(msg interface{}, peer Peer)
}

// Server is a placeholder for a p2p service. To be designed.
type Server struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mutex         *sync.Mutex
	feeds         map[reflect.Type]Feed
	host          host.Host
	dht           *kaddht.IpfsDHT
	gsub          *pubsub.PubSub
	topicMapping  map[reflect.Type]string
	bootstrapNode string
	relayNodeAddr string
}

// ServerConfig for peer to peer networking.
type ServerConfig struct {
	BootstrapNodeAddr string
	RelayNodeAddr     string
	Port              int
}

// NewServer creates a new p2p server instance.
func NewServer(cfg *ServerConfig) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := buildOptions(cfg.Port)
	if cfg.RelayNodeAddr != "" {
		opts = append(opts, libp2p.AddrsFactory(relayAddrsOnly(cfg.RelayNodeAddr)))
	}
	if !checkAvailablePort(cfg.Port) {
		cancel()
		return nil, fmt.Errorf("error listening on p2p, port %d already taken", cfg.Port)
	}
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
		ctx:           ctx,
		cancel:        cancel,
		feeds:         make(map[reflect.Type]Feed),
		host:          h,
		dht:           dht,
		gsub:          gsub,
		mutex:         &sync.Mutex{},
		topicMapping:  make(map[reflect.Type]string),
		bootstrapNode: cfg.BootstrapNodeAddr,
		relayNodeAddr: cfg.RelayNodeAddr,
	}, nil
}

func checkAvailablePort(port int) bool {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Errorf("Could not get IPv4 address: %v", err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return false
	}

	if err := ln.Close(); err != nil {
		log.Errorf("Could not close listener %v", err)
	}

	return true
}

// Start the main routine for an p2p server.
func (s *Server) Start() {
	ctx, span := trace.StartSpan(s.ctx, "p2p_server_start")
	defer span.End()
	log.Info("Starting service")

	if s.bootstrapNode != "" {
		if err := startDHTDiscovery(ctx, s.host, s.bootstrapNode); err != nil {
			log.Errorf("Could not start peer discovery via DHT: %v", err)
		}
		if err := s.dht.Bootstrap(ctx); err != nil {
			log.Errorf("Failed to bootstrap DHT: %v", err)
		}
	}

	if s.relayNodeAddr != "" {
		if err := dialRelayNode(ctx, s.host, s.relayNodeAddr); err != nil {
			log.Errorf("Could not dial relay node: %v", err)
		}
	}

	if err := startmDNSDiscovery(ctx, s.host); err != nil {
		log.Errorf("Could not start peer discovery via mDNS: %v", err)
		return
	}

	startPeerWatcher(ctx, s.host)
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	log.Info("Stopping service")

	s.cancel()
	return nil
}

// Status returns an error if the p2p service does not have sufficient peers.
func (s *Server) Status() error {
	if peerCount(s.host) < 5 {
		return errors.New("less than 5 peers")
	}
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

			d := message
			if err := proto.Unmarshal(msg.Data, d); err != nil {
				log.WithError(err).Error("Failed to decode data")
				continue
			}

			var h Handler = func(pMsg Message) {
				s.emit(pMsg, feed)
			}

			pMsg := Message{Ctx: s.ctx, Data: d}
			for _, adapter := range adapters {
				h = adapter(h)
			}

			h(pMsg)
		}
	}()
}

func (s *Server) emit(msg Message, feed Feed) {
	i := feed.Send(msg)
	log.WithFields(logrus.Fields{
		"numSubs": i,
		"msgType": fmt.Sprintf("%T", msg.Data),
		"msgName": proto.MessageName(msg.Data),
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

// Broadcast publishes a message to all localized peers using gossipsub.
// msg must be a proto.Message that can be encoded into a byte array.
// It publishes the first 100 chars of msg over the msg's mapped topic.
// To map a messageType to a topic, use RegisterTopic.
//
// It logs an error if msg is not a protobuf message,
// if msg cannot be encoded into a byte array,
// or if the server is unable to publish the message over gossipsub.
//
//   msg := make(chan p2p.Message, 100) // Choose a reasonable buffer size!
//   ps.RegisterTopic("message_topic_here", msg)
//   ps.Broadcast(msg)
func (s *Server) Broadcast(msg proto.Message) {
	topic := s.topicMapping[messageType(msg)]

	// Shorten message if it is too long to avoid
	// polluting the logs.
	if len(msg.String()) > 100 {
		newMessage := msg.String()[:100]

		log.WithFields(logrus.Fields{
			"topic": topic,
		}).Debugf("Broadcasting msg %+v --Message too long to be displayed", newMessage)

	} else {
		log.WithFields(logrus.Fields{
			"topic": topic,
		}).Debugf("Broadcasting msg %+v", msg)
	}

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
