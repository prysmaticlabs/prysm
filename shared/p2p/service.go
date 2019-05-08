package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	ggio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const prysmProtocolPrefix = "/prysm/0.0.0"

// We accommodate p2p message sizes as large as ~17Mb as we are transmitting
// full beacon states over the wire for our current implementation.
const maxMessageSize = 1 << 24

// Sender represents a struct that is able to relay information via p2p.
// Server implements this interface.
type Sender interface {
	Send(ctx context.Context, msg proto.Message, peer peer.ID) error
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
	noDiscovery   bool
}

// ServerConfig for peer to peer networking.
type ServerConfig struct {
	NoDiscovery            bool
	BootstrapNodeAddr      string
	RelayNodeAddr          string
	HostAddress            string
	Port                   int
	DepositContractAddress string
}

// NewServer creates a new p2p server instance.
func NewServer(cfg *ServerConfig) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := buildOptions(cfg.Port)
	if cfg.RelayNodeAddr != "" {
		opts = append(opts, libp2p.AddrsFactory(withRelayAddrs(cfg.RelayNodeAddr)))
	} else if cfg.HostAddress != "" {
		opts = append(opts, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			external, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.HostAddress, cfg.Port))
			if err != nil {
				log.WithError(err).Error("Unable to create external multiaddress")
			} else {
				addrs = append(addrs, external)
			}
			return addrs
		}))
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

	psOpts := []pubsub.Option{
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
	}
	var gsub *pubsub.PubSub
	if featureconfig.FeatureConfig().DisableGossipSub {
		gsub, err = pubsub.NewFloodSub(ctx, h, psOpts...)
	} else {
		gsub, err = pubsub.NewGossipSub(ctx, h, psOpts...)
	}
	if err != nil {
		cancel()
		return nil, err
	}

	// Blockchain peering negotiation; excludes negotiating with bootstrap or
	// relay nodes.
	exclusions := []peer.ID{}
	for _, addr := range []string{cfg.BootstrapNodeAddr, cfg.RelayNodeAddr} {
		if addr == "" {
			continue
		}
		info, err := peerInfoFromAddr(addr)
		if err != nil {
			return nil, err
		}
		exclusions = append(exclusions, info.ID)
	}
	setupPeerNegotiation(h, cfg.DepositContractAddress, exclusions)
	setHandshakeHandler(h, cfg.DepositContractAddress)

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
		noDiscovery:   cfg.NoDiscovery,
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

	if !s.noDiscovery && s.bootstrapNode != "" {
		if err := startDHTDiscovery(ctx, s.host, s.bootstrapNode); err != nil {
			log.Errorf("Could not start peer discovery via DHT: %v", err)
		}
		bcfg := kaddht.DefaultBootstrapConfig
		bcfg.Period = time.Duration(30 * time.Second)
		if err := s.dht.BootstrapWithConfig(ctx, bcfg); err != nil {
			log.Errorf("Failed to bootstrap DHT: %v", err)
		}
	}
	if !s.noDiscovery && s.relayNodeAddr != "" {
		if err := dialRelayNode(ctx, s.host, s.relayNodeAddr); err != nil {
			log.Errorf("Could not dial relay node: %v", err)
		}
	}

	if err := startmDNSDiscovery(ctx, s.host); err != nil {
		log.Errorf("Could not start peer discovery via mDNS: %v", err)
		return
	}

	if !s.noDiscovery {
		startPeerWatcher(ctx, s.host, s.bootstrapNode, s.relayNodeAddr)
	}
}

// Stop the main p2p loop.
func (s *Server) Stop() error {
	log.Info("Stopping service")

	s.cancel()
	return nil
}

// Status returns an error if the p2p service does not have sufficient peers.
func (s *Server) Status() error {
	if peerCount(s.host) < 3 {
		return errors.New("less than 3 peers")
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

	handler := func(msg *pb.Envelope, peerID peer.ID) {
		log.WithField("topic", topic).Debug("Processing incoming message")
		var h Handler = func(pMsg Message) {
			s.emit(pMsg, feed)
		}

		ctx := context.Background()

		spanCtx, ok := propagation.FromBinary(msg.SpanContext)
		if !ok {
			log.Error("Invalid span context from p2p message")
			return
		}
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "beacon-chain.p2p.receiveMessage", spanCtx)
		defer span.End()
		span.AddAttributes(
			trace.StringAttribute("topic", topic),
			trace.StringAttribute("peerID", peerID.String()),
		)

		data := proto.Clone(message)
		if err := proto.Unmarshal(msg.Payload, data); err != nil {
			log.Error("Could not unmarshal payload")
		}
		pMsg := Message{Ctx: ctx, Data: data, Peer: peerID}
		for _, adapter := range adapters {
			h = adapter(h)
		}

		h(pMsg)
	}

	s.host.SetStreamHandler(protocol.ID(prysmProtocolPrefix+"/"+topic), func(stream libp2pnet.Stream) {
		log.WithField("topic", topic).Debug("Received new stream")
		defer stream.Close()
		r := ggio.NewDelimitedReader(stream, maxMessageSize)
		defer r.Close()

		msg := &pb.Envelope{}
		for {
			err := r.ReadMsg(msg)
			if err == io.EOF {
				return // end of stream
			}
			if err != nil {
				log.WithError(err).Error("Could not read message from stream")
				return
			}

			handler(msg, stream.Conn().RemotePeer())
		}
	})

	go func() {
		defer sub.Cancel()

		var msg *pubsub.Message
		var err error

		// Recover from any panic as part of the receive p2p msg process.
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(logrus.Fields{
					"r":        r,
					"msg.Data": attemptToConvertPbToString(msg.Data, message),
				}).Error("P2P message caused a panic! Recovering...")
			}
		}()

		for {
			msg, err = sub.Next(s.ctx)

			if s.ctx.Err() != nil {
				log.WithError(s.ctx.Err()).Debug("Context error")
				return
			}
			if err != nil {
				log.Errorf("Failed to get next message: %v", err)
				continue
			}

			if msg == nil || msg.GetFrom() == s.host.ID() {
				continue
			}

			d := &pb.Envelope{}
			if err := proto.Unmarshal(msg.Data, d); err != nil {
				log.WithError(err).Error("Failed to decode data")
				continue
			}

			handler(d, msg.GetFrom())
		}
	}()
}

// Attempts to convert some proto.Message to a string in a panic safe method.
func attemptToConvertPbToString(b []byte, msg proto.Message) string {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("r", r).Error("Panicked when trying to log PB")
		}
	}()
	if err := proto.Unmarshal(b, msg); err != nil {
		log.WithError(err).Error("Failed to decode data")
		return ""
	}

	return proto.MarshalTextString(msg)
}

func (s *Server) emit(msg Message, feed Feed) {
	i := feed.Send(msg)
	log.WithFields(logrus.Fields{
		"numSubs": i,
		"msgType": fmt.Sprintf("%T", msg.Data),
		"msgName": proto.MessageName(msg.Data),
	}).Debug("Emit p2p message to feed subscribers")
	if span := trace.FromContext(msg.Ctx); span != nil {
		span.AddAttributes(trace.Int64Attribute("feedSubscribers", int64(i)))
	}
}

// Subscribe returns a subscription to a feed of msg's Type and adds the channels to the feed.
func (s *Server) Subscribe(msg proto.Message, channel chan Message) event.Subscription {
	return s.Feed(msg).Subscribe(channel)
}

// Send a message to a specific peer. If the peerID is set to p2p.AnyPeer, then
// this method will act as a broadcast.
func (s *Server) Send(ctx context.Context, msg proto.Message, peerID peer.ID) error {
	isPeer := false
	for _, p := range s.host.Network().Peers() {
		if p == peerID {
			isPeer = true
			break
		}
	}

	if peerID == AnyPeer || s.host.Network().Connectedness(peerID) == libp2pnet.CannotConnect || !isPeer {
		s.Broadcast(ctx, msg)
		return nil
	}

	ctx, span := trace.StartSpan(ctx, "p2p.Send")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	topic := s.topicMapping[messageType(msg)]
	pid := protocol.ID(prysmProtocolPrefix + "/" + topic)
	stream, err := s.host.NewStream(ctx, peerID, pid)
	if err != nil {
		return err
	}
	defer stream.Close()

	w := ggio.NewDelimitedWriter(stream)
	defer w.Close()

	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	envelope := &pb.Envelope{
		SpanContext: propagation.Binary(span.SpanContext()),
		Payload:     b,
	}

	return w.WriteMsg(envelope)
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
func (s *Server) Broadcast(ctx context.Context, msg proto.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("r", r).Error("Panicked when broadcasting!")
		}
	}()

	ctx, span := trace.StartSpan(ctx, "beacon-chain.p2p.Broadcast")
	defer span.End()

	topic := s.topicMapping[messageType(msg)]
	span.AddAttributes(trace.StringAttribute("topic", topic))

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
			"msg":   msg,
		}).Debug("Broadcasting msg")
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

	envelope := &pb.Envelope{
		SpanContext: propagation.Binary(span.SpanContext()),
		Payload:     b,
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		log.Errorf("Failed to marshal data for broadcast: %v", err)
		return
	}

	if err := s.gsub.Publish(topic, data); err != nil {
		log.Errorf("Failed to publish to gossipsub topic: %v", err)
	}
}
