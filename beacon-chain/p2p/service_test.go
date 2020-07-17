package p2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockListener struct{}

func (mockListener) Self() *enode.Node {
	panic("implement me")
}

func (mockListener) Close() {
	//no-op
}

func (mockListener) Lookup(enode.ID) []*enode.Node {
	panic("implement me")
}

func (mockListener) ReadRandomNodes([]*enode.Node) int {
	panic("implement me")
}

func (mockListener) Resolve(*enode.Node) *enode.Node {
	panic("implement me")
}

func (mockListener) Ping(*enode.Node) error {
	panic("implement me")
}

func (mockListener) RequestENR(*enode.Node) (*enode.Node, error) {
	panic("implement me")
}

func (mockListener) LocalNode() *enode.LocalNode {
	panic("implement me")
}

func (mockListener) RandomNodes() enode.Iterator {
	panic("implement me")
}

func createHost(t *testing.T, port int) (host.Host, *ecdsa.PrivateKey, net.IP) {
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr = net.ParseIP("127.0.0.1")
	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, port))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	return h, pkey, ipAddr
}

func TestService_Stop_SetsStartedToFalse(t *testing.T) {
	s, err := NewService(&Config{})
	require.NoError(t, err)
	s.started = true
	s.dv5Listener = &mockListener{}
	assert.NoError(t, s.Stop())
	assert.Equal(t, false, s.started)
}

func TestService_Stop_DontPanicIfDv5ListenerIsNotInited(t *testing.T) {
	s, err := NewService(&Config{})
	require.NoError(t, err)
	assert.NoError(t, s.Stop())
}

func TestService_Start_OnlyStartsOnce(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := &Config{
		TCPPort: 2000,
		UDPPort: 2000,
	}
	s, err := NewService(cfg)
	require.NoError(t, err)
	s.stateNotifier = &mock.MockStateNotifier{}
	s.dv5Listener = &mockListener{}
	exitRoutine := make(chan bool)
	go func() {
		s.Start()
		<-exitRoutine
	}()
	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Initialized,
			Data: &statefeed.InitializedData{
				StartTime:             time.Now(),
				GenesisValidatorsRoot: make([]byte, 32),
			},
		})
	}
	time.Sleep(time.Second * 2)
	assert.Equal(t, true, s.started, "Expected service to be started")
	s.Start()
	testutil.AssertLogsContain(t, hook, "Attempted to start p2p service when it was already started")
	require.NoError(t, s.Stop())
	exitRoutine <- true
}

func TestService_Status_NotRunning(t *testing.T) {
	s := &Service{started: false}
	s.dv5Listener = &mockListener{}
	assert.ErrorContains(t, "not running", s.Status(), "Status returned wrong error")
}

func TestListenForNewNodes(t *testing.T) {
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

	var listeners []*discover.UDPv5
	var hosts []host.Host
	// setup other nodes.
	cfg = &Config{
		BootstrapNodeAddr:   []string{bootNode.String()},
		Discv5BootStrapAddr: []string{bootNode.String()},
		MaxPeers:            30,
	}
	for i := 1; i <= 5; i++ {
		h, pkey, ipAddr := createHost(t, port+i)
		cfg.UDPPort = uint(port + i)
		cfg.TCPPort = uint(port + i)
		s := &Service{
			cfg:                   cfg,
			genesisTime:           genesisTime,
			genesisValidatorsRoot: genesisValidatorsRoot,
		}
		listener, err := s.startDiscoveryV5(ipAddr, pkey)
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

	cfg.UDPPort = 14000
	cfg.TCPPort = 14001

	s, err = NewService(cfg)
	require.NoError(t, err)
	s.stateNotifier = &mock.MockStateNotifier{}
	exitRoutine := make(chan bool)
	go func() {
		s.Start()
		<-exitRoutine
	}()
	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Initialized,
			Data: &statefeed.InitializedData{
				StartTime:             genesisTime,
				GenesisValidatorsRoot: genesisValidatorsRoot,
			},
		})
	}
	time.Sleep(4 * time.Second)
	peers := s.host.Network().Peers()
	assert.Equal(t, 5, len(peers), "Not all peers added to peerstore")
	require.NoError(t, s.Stop())
	exitRoutine <- true
}

func TestPeer_Disconnect(t *testing.T) {
	h1, _, _ := createHost(t, 5000)
	defer func() {
		if err := h1.Close(); err != nil {
			t.Log(err)
		}
	}()

	s := &Service{
		host: h1,
	}

	h2, _, ipaddr := createHost(t, 5001)
	defer func() {
		if err := h2.Close(); err != nil {
			t.Log(err)
		}
	}()

	h2Addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipaddr, 5001, h2.ID()))
	require.NoError(t, err)
	addrInfo, err := peer.AddrInfoFromP2pAddr(h2Addr)
	require.NoError(t, err)
	require.NoError(t, s.host.Connect(context.Background(), *addrInfo))
	assert.Equal(t, 1, len(s.host.Network().Peers()), "Invalid number of peers")
	assert.Equal(t, 1, len(s.host.Network().Conns()), "Invalid number of connections")
	require.NoError(t, s.Disconnect(h2.ID()))
	assert.Equal(t, 0, len(s.host.Network().Conns()), "Invalid number of connections")
}

func TestService_JoinLeaveTopic(t *testing.T) {
	s, err := NewService(&Config{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(s.joinedTopics))

	topic := fmt.Sprintf(AttestationSubnetTopicFormat, 42, 42)
	topicHandle, err := s.JoinTopic(topic)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(s.joinedTopics))

	sub, err := topicHandle.Subscribe()
	assert.NoError(t, err)

	// Try leaving topic that has subscriptions.
	want := "cannot close topic: outstanding event handlers or subscriptions"
	assert.ErrorContains(t, want, s.LeaveTopic(topic))

	// After subscription is cancelled, leaving topic should not result in error.
	sub.Cancel()
	assert.NoError(t, s.LeaveTopic(topic))
}
