// Package p2p defines the network protocol implementation for eth2
// used by beacon nodes, including peer discovery using discv5, gossip-sub
// using libp2p, and handing peer lifecycles + handshakes.
package p2p

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	filter "github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

var _ = shared.Service(&Service{})

// In the event that we are at our peer limit, we
// stop looking for new peers and instead poll
// for the current peer limit status for the time period
// defined below.
var pollingPeriod = 6 * time.Second

// Refresh rate of ENR set at twice per slot.
var refreshRate = slotutil.DivideSlotBy(2)

// lookup limit whenever looking up for random nodes.
const lookupLimit = 15

// maxBadResponses is the maximum number of bad responses from a peer before we stop talking to it.
const maxBadResponses = 5

// Exclusion list cache config values.
const cacheNumCounters, cacheMaxCost, cacheBufferItems = 1000, 1000, 64

// maxDialTimeout is the timeout for a single peer dial.
const maxDialTimeout = 30 * time.Second

// Service for managing peer to peer (p2p) networking.
type Service struct {
	started               bool
	isPreGenesis          bool
	pingMethod            func(ctx context.Context, id peer.ID) error
	cancel                context.CancelFunc
	cfg                   *Config
	peers                 *peers.Status
	addrFilter            *filter.Filters
	privKey               *ecdsa.PrivateKey
	exclusionList         *ristretto.Cache
	metaData              *pb.MetaData
	pubsub                *pubsub.PubSub
	joinedTopics          map[string]*pubsub.Topic
	dv5Listener           Listener
	startupErr            error
	stateNotifier         statefeed.Notifier
	ctx                   context.Context
	host                  host.Host
	genesisTime           time.Time
	genesisValidatorsRoot []byte
}

// NewService initializes a new p2p service compatible with shared.Service interface. No
// connections are made until the Start function is called during the service registry startup.
func NewService(cfg *Config) (*Service, error) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop().
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheNumCounters,
		MaxCost:     cacheMaxCost,
		BufferItems: cacheBufferItems,
	})
	if err != nil {
		return nil, err
	}

	s := &Service{
		ctx:           ctx,
		stateNotifier: cfg.StateNotifier,
		cancel:        cancel,
		cfg:           cfg,
		exclusionList: cache,
		isPreGenesis:  true,
		joinedTopics:  make(map[string]*pubsub.Topic, len(GossipTopicMappings)),
	}

	dv5Nodes := parseBootStrapAddrs(s.cfg.BootstrapNodeAddr)

	cfg.Discv5BootStrapAddr = dv5Nodes

	ipAddr := ipAddr()
	s.privKey, err = privKey(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to generate p2p private key")
		return nil, err
	}
	s.metaData, err = metaDataFromConfig(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to create peer metadata")
		return nil, err
	}
	s.addrFilter, err = configureFilter(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to create address filter")
		return nil, err
	}
	opts := s.buildOptions(ipAddr, s.privKey)
	h, err := libp2p.New(s.ctx, opts...)
	if err != nil {
		log.WithError(err).Error("Failed to create p2p host")
		return nil, err
	}

	s.host = h

	// Gossipsub registration is done before we add in any new peers
	// due to libp2p's gossipsub implementation not taking into
	// account previously added peers when creating the gossipsub
	// object.
	psOpts := []pubsub.Option{
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
		pubsub.WithMessageIdFn(msgIDFunction),
	}

	gs, err := pubsub.NewGossipSub(s.ctx, s.host, psOpts...)
	if err != nil {
		log.WithError(err).Error("Failed to start pubsub")
		return nil, err
	}
	s.pubsub = gs

	s.peers = peers.NewStatus(ctx, &peers.StatusParams{
		PeerLimit: int(s.cfg.MaxPeers),
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold:     maxBadResponses,
			BadResponsesWeight:        -100,
			BadResponsesDecayInterval: time.Hour,
		},
	})

	return s, nil
}

// Start the p2p service.
func (s *Service) Start() {
	if s.started {
		log.Error("Attempted to start p2p service when it was already started")
		return
	}

	// Waits until the state is initialized via an event feed.
	// Used for fork-related data when connecting peers.
	s.awaitStateInitialized()
	s.isPreGenesis = false

	var peersToWatch []string
	if s.cfg.RelayNodeAddr != "" {
		peersToWatch = append(peersToWatch, s.cfg.RelayNodeAddr)
		if err := dialRelayNode(s.ctx, s.host, s.cfg.RelayNodeAddr); err != nil {
			log.WithError(err).Errorf("Could not dial relay node")
		}
	}

	if !s.cfg.NoDiscovery && !s.cfg.DisableDiscv5 {
		ipAddr := ipAddr()
		listener, err := s.startDiscoveryV5(
			ipAddr,
			s.privKey,
		)
		if err != nil {
			log.WithError(err).Fatal("Failed to start discovery")
			s.startupErr = err
			return
		}
		err = s.connectToBootnodes()
		if err != nil {
			log.WithError(err).Error("Could not add bootnode to the exclusion list")
			s.startupErr = err
			return
		}
		s.dv5Listener = listener
		go s.listenForNewNodes()
	}

	s.started = true

	if len(s.cfg.StaticPeers) > 0 {
		addrs, err := peersFromStringAddrs(s.cfg.StaticPeers)
		if err != nil {
			log.Errorf("Could not connect to static peer: %v", err)
		}
		s.connectWithAllPeers(addrs)
	}

	// Periodic functions.
	runutil.RunEvery(s.ctx, params.BeaconNetworkConfig().TtfbTimeout, func() {
		ensurePeerConnections(s.ctx, s.host, peersToWatch...)
	})
	runutil.RunEvery(s.ctx, 30*time.Minute, s.Peers().Prune)
	runutil.RunEvery(s.ctx, params.BeaconNetworkConfig().RespTimeout, s.updateMetrics)
	runutil.RunEvery(s.ctx, refreshRate, func() {
		s.RefreshENR()
	})

	multiAddrs := s.host.Network().ListenAddresses()
	logIPAddr(s.host.ID(), multiAddrs...)

	p2pHostAddress := s.cfg.HostAddress
	p2pTCPPort := s.cfg.TCPPort

	if p2pHostAddress != "" {
		logExternalIPAddr(s.host.ID(), p2pHostAddress, p2pTCPPort)
		verifyConnectivity(p2pHostAddress, p2pTCPPort, "tcp")
	}

	p2pHostDNS := s.cfg.HostDNS
	if p2pHostDNS != "" {
		logExternalDNSAddr(s.host.ID(), p2pHostDNS, p2pTCPPort)
	}
}

// Stop the p2p service and terminate all peer connections.
func (s *Service) Stop() error {
	defer s.cancel()
	s.started = false
	if s.dv5Listener != nil {
		s.dv5Listener.Close()
	}
	return nil
}

// Status of the p2p service. Will return an error if the service is considered unhealthy to
// indicate that this node should not serve traffic until the issue has been resolved.
func (s *Service) Status() error {
	if s.isPreGenesis {
		return nil
	}
	if !s.started {
		return errors.New("not running")
	}
	if s.startupErr != nil {
		return s.startupErr
	}
	return nil
}

// Started returns true if the p2p service has successfully started.
func (s *Service) Started() bool {
	return s.started
}

// Encoding returns the configured networking encoding.
func (s *Service) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{}
}

// PubSub returns the p2p pubsub framework.
func (s *Service) PubSub() *pubsub.PubSub {
	return s.pubsub
}

// Host returns the currently running libp2p
// host of the service.
func (s *Service) Host() host.Host {
	return s.host
}

// SetStreamHandler sets the protocol handler on the p2p host multiplexer.
// This method is a pass through to libp2pcore.Host.SetStreamHandler.
func (s *Service) SetStreamHandler(topic string, handler network.StreamHandler) {
	s.host.SetStreamHandler(protocol.ID(topic), handler)
}

// PeerID returns the Peer ID of the local peer.
func (s *Service) PeerID() peer.ID {
	return s.host.ID()
}

// Disconnect from a peer.
func (s *Service) Disconnect(pid peer.ID) error {
	return s.host.Network().ClosePeer(pid)
}

// Connect to a specific peer.
func (s *Service) Connect(pi peer.AddrInfo) error {
	return s.host.Connect(s.ctx, pi)
}

// Peers returns the peer status interface.
func (s *Service) Peers() *peers.Status {
	return s.peers
}

// ENR returns the local node's current ENR.
func (s *Service) ENR() *enr.Record {
	if s.dv5Listener == nil {
		return nil
	}
	return s.dv5Listener.Self().Record()
}

// Metadata returns a copy of the peer's metadata.
func (s *Service) Metadata() *pb.MetaData {
	return proto.Clone(s.metaData).(*pb.MetaData)
}

// MetadataSeq returns the metadata sequence number.
func (s *Service) MetadataSeq() uint64 {
	return s.metaData.SeqNumber
}

// AddPingMethod adds the metadata ping rpc method to the p2p service, so that it can
// be used to refresh ENR.
func (s *Service) AddPingMethod(reqFunc func(ctx context.Context, id peer.ID) error) {
	s.pingMethod = reqFunc
}

func (s *Service) pingPeers() {
	if s.pingMethod == nil {
		return
	}
	for _, pid := range s.peers.Connected() {
		go func(id peer.ID) {
			if err := s.pingMethod(s.ctx, id); err != nil {
				log.WithField("peer", id).WithError(err).Error("Failed to ping peer")
			}
		}(pid)
	}
}

// Waits for the beacon state to be initialized, important
// for initializing the p2p service as p2p needs to be aware
// of genesis information for peering.
func (s *Service) awaitStateInitialized() {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	for {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.Initialized {
				data, ok := event.Data.(*statefeed.InitializedData)
				if !ok {
					log.Fatalf("Received wrong data over state initialized feed: %v", data)
				}
				s.genesisTime = data.StartTime
				s.genesisValidatorsRoot = data.GenesisValidatorsRoot
				return
			}
		}
	}
}

func (s *Service) connectWithAllPeers(multiAddrs []ma.Multiaddr) {
	addrInfos, err := peer.AddrInfosFromP2pAddrs(multiAddrs...)
	if err != nil {
		log.Errorf("Could not convert to peer address info's from multiaddresses: %v", err)
		return
	}
	for _, info := range addrInfos {
		// make each dial non-blocking
		go func(info peer.AddrInfo) {
			if err := s.connectWithPeer(info); err != nil {
				log.WithError(err).Tracef("Could not connect with peer %s", info.String())
			}
		}(info)
	}
}

func (s *Service) connectWithPeer(info peer.AddrInfo) error {
	if info.ID == s.host.ID() {
		return nil
	}
	if s.Peers().IsBad(info.ID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(s.ctx, maxDialTimeout)
	defer cancel()
	if err := s.host.Connect(ctx, info); err != nil {
		s.Peers().Scorer().IncrementBadResponses(info.ID)
		return err
	}
	return nil
}

func (s *Service) connectToBootnodes() error {
	nodes := make([]*enode.Node, 0, len(s.cfg.Discv5BootStrapAddr))
	for _, addr := range s.cfg.Discv5BootStrapAddr {
		bootNode, err := enode.Parse(enode.ValidSchemes, addr)
		if err != nil {
			return err
		}
		// do not dial bootnodes with their tcp ports not set
		if err := bootNode.Record().Load(enr.WithEntry("tcp", new(enr.TCP))); err != nil {
			if !enr.IsNotFound(err) {
				log.WithError(err).Error("Could not retrieve tcp port")
			}
			continue
		}
		nodes = append(nodes, bootNode)
	}
	multiAddresses := convertToMultiAddr(nodes)
	s.connectWithAllPeers(multiAddresses)
	return nil
}
