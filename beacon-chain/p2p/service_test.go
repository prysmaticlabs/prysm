package p2p

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/multiformats/go-multiaddr"
	testing2 "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockListener struct{}

func (m *mockListener) Self() *discv5.Node {
	panic("implement me")
}

func (m *mockListener) Close() {
	//no-op
}

func (m *mockListener) Lookup(discv5.NodeID) []*discv5.Node {
	panic("implement me")
}

func (m *mockListener) ReadRandomNodes([]*discv5.Node) int {
	panic("implement me")
}

func (m *mockListener) SetFallbackNodes([]*discv5.Node) error {
	panic("implement me")
}

func (m *mockListener) Resolve(discv5.NodeID) *discv5.Node {
	panic("implement me")
}

func (m *mockListener) RegisterTopic(discv5.Topic, <-chan struct{}) {
	panic("implement me")
}

func (m *mockListener) SearchTopic(discv5.Topic, <-chan time.Duration, chan<- *discv5.Node, chan<- bool) {
	panic("implement me")
}

func tcpPortFromHost(t *testing.T, host host.Host) int64 {
	pinfo := host.Peerstore().PeerInfo(host.ID())
	addresses := multiaddr.Split(pinfo.Addrs[0])
	tcpAddr := addresses[len(addresses)-1]
	trimmedStr := strings.Trim(tcpAddr.String(), "/tcp/")
	port, err := strconv.ParseInt(trimmedStr, 10, 64)
	if err != nil {
		t.Fatalf("Could not get tcp port from peer: %v", err)
	}
	return port
}

func createPeer(t *testing.T, cfg *Config) (Listener, *testing2.TestP2P) {
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr = net.ParseIP("127.0.0.1")
	privKey := convertToInterfacePrivkey(pkey)
	peerNode := testing2.NewTestP2PWithKey(t, privKey)
	port := tcpPortFromHost(t, peerNode.Host)

	cfg.UDPPort = uint(port)
	cfg.Port = uint(port)
	listener, err := startDiscoveryV5(ipAddr, pkey, cfg)
	if err != nil {
		t.Errorf("Could not start discovery for node: %v", err)
	}
	return listener, peerNode
}

func TestService_Stop_SetsStartedToFalse(t *testing.T) {
	s, _ := NewService(nil)
	s.started = true
	s.dv5Listener = &mockListener{}
	_ = s.Stop()

	if s.started != false {
		t.Error("Expected Service.started to be false, got true")
	}
}

func TestService_Start_OnlyStartsOnce(t *testing.T) {
	hook := logTest.NewGlobal()

	s, _ := NewService(&Config{})
	s.dv5Listener = &mockListener{}
	defer s.Stop()
	s.Start()
	if s.started != true {
		t.Error("Expected service to be started")
	}
	s.Start()
	testutil.AssertLogsContain(t, hook, "Attempted to start p2p service when it was already started")
}

func TestService_Status_NotRunning(t *testing.T) {
	s := &Service{started: false}
	s.dv5Listener = &mockListener{}
	if s.Status().Error() != "not running" {
		t.Errorf("Status returned wrong error, got %v", s.Status())
	}
}

func TestListenForNewNodes(t *testing.T) {
	// setup bootnode
	port := 2000
	_, pkey := createAddrAndPrivKey(t)
	ipAddr := net.ParseIP("127.0.0.1")
	bootListener := createListener(ipAddr, port, pkey)
	defer bootListener.Close()

	bootNode := bootListener.Self()

	cfg := &Config{
		BootstrapNodeAddr: bootNode.String(),
	}
	// setup other nodes
	for i := 1; i <= 5; i++ {
		_, _ = createPeer(t, cfg)
	}
	listener, testp2p := createPeer(t, cfg)

	s := &Service{
		dv5Listener: listener,
		host:        testp2p.Host,
		pubsub:      testp2p.PubSub(),
		cfg:         cfg,
		ctx:         context.Background(),
	}

	go s.listenForNewNodes()

	time.Sleep(2 * time.Second)
	peers := testp2p.Host.Network().Peers()
	if len(peers) != 5 {
		t.Errorf("Not all peers added to peerstore, wanted %d but got %d", 5, len(peers))
	}
}
