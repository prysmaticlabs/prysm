package p2p

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_Broadcast(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) == 0 {
		t.Fatal("No peers")
	}

	p := &Service{
		host:                  p1.BHost,
		pubsub:                p1.PubSub(),
		joinedTopics:          map[string]*pubsub.Topic{},
		cfg:                   &Config{},
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
	}

	msg := &testpb.TestSimpleMessage{
		Bar: 55,
	}

	topic := "/eth2/%x/testing"
	// Set a test gossip mapping for testpb.TestSimpleMessage.
	GossipTypeMapping[reflect.TypeOf(msg)] = topic
	digest, err := p.forkDigest()
	require.NoError(t, err)
	topic = fmt.Sprintf(topic, digest)

	// External peer subscribes to the topic.
	topic += p.Encoding().ProtocolSuffix()
	sub, err := p2.SubscribeToTopic(topic)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond) // libp2p fails without this delay...

	// Async listen for the pubsub, must be before the broadcast.
	var wg sync.WaitGroup
	wg.Add(1)
	go func(tt *testing.T) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		incomingMessage, err := sub.Next(ctx)
		require.NoError(t, err)

		result := &testpb.TestSimpleMessage{}
		require.NoError(t, p.Encoding().DecodeGossip(incomingMessage.Data, result))
		if !proto.Equal(result, msg) {
			tt.Errorf("Did not receive expected message, got %+v, wanted %+v", result, msg)
		}
	}(t)

	// Broadcast to peers and wait.
	require.NoError(t, p.Broadcast(context.Background(), msg))
	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Error("Failed to receive pubsub within 1s")
	}
}

func TestService_Broadcast_ReturnsErr_TopicNotMapped(t *testing.T) {
	p := Service{
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
	}
	assert.ErrorContains(t, ErrMessageNotMapped.Error(), p.Broadcast(context.Background(), &testpb.AddressBook{}))
}

func TestService_Attestation_Subnet(t *testing.T) {
	if gtm := GossipTypeMapping[reflect.TypeOf(&eth.Attestation{})]; gtm != AttestationSubnetTopicFormat {
		t.Errorf("Constant is out of date. Wanted %s, got %s", AttestationSubnetTopicFormat, gtm)
	}

	tests := []struct {
		att   *eth.Attestation
		topic string
	}{
		{
			att: &eth.Attestation{
				Data: &eth.AttestationData{
					CommitteeIndex: 0,
					Slot:           2,
				},
			},
			topic: "/eth2/00000000/beacon_attestation_2",
		},
		{
			att: &eth.Attestation{
				Data: &eth.AttestationData{
					CommitteeIndex: 11,
					Slot:           10,
				},
			},
			topic: "/eth2/00000000/beacon_attestation_21",
		},
		{
			att: &eth.Attestation{
				Data: &eth.AttestationData{
					CommitteeIndex: 55,
					Slot:           529,
				},
			},
			topic: "/eth2/00000000/beacon_attestation_8",
		},
	}
	for _, tt := range tests {
		subnet := helpers.ComputeSubnetFromCommitteeAndSlot(100, tt.att.Data.CommitteeIndex, tt.att.Data.Slot)
		assert.Equal(t, tt.topic, attestationToTopic(subnet, [4]byte{} /* fork digest */), "Wrong topic")
	}
}

func TestService_BroadcastAttestation(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) == 0 {
		t.Fatal("No peers")
	}

	p := &Service{
		host:                  p1.BHost,
		pubsub:                p1.PubSub(),
		joinedTopics:          map[string]*pubsub.Topic{},
		cfg:                   &Config{},
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
		subnetsLock:           make(map[uint64]*sync.RWMutex),
		subnetsLockLock:       sync.Mutex{},
		peers: peers.NewStatus(context.Background(), &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		}),
	}

	msg := &eth.Attestation{
		AggregationBits: bitfield.NewBitlist(7),
		Data: &eth.AttestationData{
			Slot:            0,
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Source: &eth.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
			Target: &eth.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
		},
		Signature: make([]byte, 96),
	}

	subnet := uint64(5)

	topic := AttestationSubnetTopicFormat
	GossipTypeMapping[reflect.TypeOf(msg)] = topic
	digest, err := p.forkDigest()
	require.NoError(t, err)
	topic = fmt.Sprintf(topic, digest, subnet)

	// External peer subscribes to the topic.
	topic += p.Encoding().ProtocolSuffix()
	sub, err := p2.SubscribeToTopic(topic)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond) // libp2p fails without this delay...

	// Async listen for the pubsub, must be before the broadcast.
	var wg sync.WaitGroup
	wg.Add(1)
	go func(tt *testing.T) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		incomingMessage, err := sub.Next(ctx)
		require.NoError(t, err)

		result := &eth.Attestation{}
		require.NoError(t, p.Encoding().DecodeGossip(incomingMessage.Data, result))
		if !proto.Equal(result, msg) {
			tt.Errorf("Did not receive expected message, got %+v, wanted %+v", result, msg)
		}
	}(t)

	// Broadcast to peers and wait.
	require.NoError(t, p.BroadcastAttestation(context.Background(), subnet, msg))
	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Error("Failed to receive pubsub within 1s")
	}
}

func TestService_BroadcastAttestationWithDiscoveryAttempts(t *testing.T) {
	// Setup bootnode.
	cfg := &Config{}
	port := 2000
	cfg.UDPPort = uint(port)
	_, pkey := createAddrAndPrivKey(t)
	ipAddr := net.ParseIP("127.0.0.1")
	genesisTime := time.Now()
	genesisValidatorsRoot := make([]byte, 32)
	s := &Service{
		cfg:                   cfg,
		genesisTime:           genesisTime,
		genesisValidatorsRoot: genesisValidatorsRoot,
	}
	bootListener, err := s.createListener(ipAddr, pkey)
	require.NoError(t, err)
	defer bootListener.Close()

	// Use shorter period for testing.
	currentPeriod := pollingPeriod
	pollingPeriod = 1 * time.Second
	defer func() {
		pollingPeriod = currentPeriod
	}()

	bootNode := bootListener.Self()
	subnet := uint64(5)

	var listeners []*discover.UDPv5
	var hosts []host.Host
	// setup other nodes.
	cfg = &Config{
		BootstrapNodeAddr:   []string{bootNode.String()},
		Discv5BootStrapAddr: []string{bootNode.String()},
		MaxPeers:            30,
	}
	// Setup 2 different hosts
	for i := 1; i <= 2; i++ {
		h, pkey, ipAddr := createHost(t, port+i)
		cfg.UDPPort = uint(port + i)
		cfg.TCPPort = uint(port + i)
		s := &Service{
			cfg:                   cfg,
			genesisTime:           genesisTime,
			genesisValidatorsRoot: genesisValidatorsRoot,
		}
		listener, err := s.startDiscoveryV5(ipAddr, pkey)
		// Set for 2nd peer
		if i == 2 {
			s.dv5Listener = listener
			s.metaData = new(pb.MetaData)
			bitV := bitfield.NewBitvector64()
			bitV.SetBitAt(subnet, true)
			s.updateSubnetRecordWithMetadata(bitV)
		}
		assert.NoError(t, err, "Could not start discovery for node")
		listeners = append(listeners, listener)
		hosts = append(hosts, h)
	}
	defer func() {
		// Close down all peers.
		for _, listener := range listeners {
			listener.Close()
		}
	}()

	// close peers upon exit of test
	defer func() {
		for _, h := range hosts {
			if err := h.Close(); err != nil {
				t.Log(err)
			}
		}
	}()

	f := featureconfig.Get()
	f.EnableAttBroadcastDiscoveryAttempts = true
	rst := featureconfig.InitWithReset(f)
	defer rst()

	ps1, err := pubsub.NewFloodSub(context.Background(), hosts[0],
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
	)
	require.NoError(t, err)

	ps2, err := pubsub.NewFloodSub(context.Background(), hosts[1],
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
	)
	require.NoError(t, err)

	p := &Service{
		host:                  hosts[0],
		pubsub:                ps1,
		dv5Listener:           listeners[0],
		joinedTopics:          map[string]*pubsub.Topic{},
		cfg:                   &Config{},
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
		subnetsLock:           make(map[uint64]*sync.RWMutex),
		subnetsLockLock:       sync.Mutex{},
		peers: peers.NewStatus(context.Background(), &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		}),
	}

	p2 := &Service{
		host:                  hosts[1],
		pubsub:                ps2,
		dv5Listener:           listeners[1],
		joinedTopics:          map[string]*pubsub.Topic{},
		cfg:                   &Config{},
		genesisTime:           time.Now(),
		genesisValidatorsRoot: []byte{'A'},
		subnetsLock:           make(map[uint64]*sync.RWMutex),
		subnetsLockLock:       sync.Mutex{},
		peers: peers.NewStatus(context.Background(), &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		}),
	}

	msg := &eth.Attestation{
		AggregationBits: bitfield.NewBitlist(7),
		Data: &eth.AttestationData{
			Slot:            0,
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Source: &eth.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
			Target: &eth.Checkpoint{
				Epoch: 0,
				Root:  make([]byte, 32),
			},
		},
		Signature: make([]byte, 96),
	}

	topic := AttestationSubnetTopicFormat
	GossipTypeMapping[reflect.TypeOf(msg)] = topic
	digest, err := p.forkDigest()
	require.NoError(t, err)
	topic = fmt.Sprintf(topic, digest, subnet)

	// External peer subscribes to the topic.
	topic += p.Encoding().ProtocolSuffix()
	sub, err := p2.SubscribeToTopic(topic)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond) // libp2p fails without this delay...

	// Async listen for the pubsub, must be before the broadcast.
	var wg sync.WaitGroup
	wg.Add(1)
	go func(tt *testing.T) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		incomingMessage, err := sub.Next(ctx)
		require.NoError(t, err)

		result := &eth.Attestation{}
		require.NoError(t, p.Encoding().DecodeGossip(incomingMessage.Data, result))
		if !proto.Equal(result, msg) {
			tt.Errorf("Did not receive expected message, got %+v, wanted %+v", result, msg)
		}
	}(t)

	// Broadcast to peers and wait.
	require.NoError(t, p.BroadcastAttestation(context.Background(), subnet, msg))
	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Error("Failed to receive pubsub within 5s")
	}
}
