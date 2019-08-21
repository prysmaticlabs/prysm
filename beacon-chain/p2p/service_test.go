package p2p

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
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

func createPeer(t *testing.T, cfg *Config, port int) (Listener, host.Host) {
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr = net.ParseIP("127.0.0.1")
	convertedKey := convertToInterfacePrivkey(pkey)
	_, err := peer.IDFromPrivateKey(convertedKey)
	if err != nil {
		t.Fatal(err)
	}
	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, port))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	cfg.UDPPort = uint(port)
	cfg.Port = uint(port)
	listener, err := startDiscoveryV5(ipAddr, pkey, cfg)
	if err != nil {
		t.Errorf("Could not start discovery for node: %v", err)
	}
	return listener, h
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

	cfg := &Config{
		Port:    2000,
		UDPPort: 2000,
	}
	s, _ := NewService(cfg)
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
	var listeners []*discv5.Network
	// setup other nodes
	for i := 1; i <= 5; i++ {
		listener, _ := createPeer(t, cfg, port+i)
		listeners = append(listeners, listener.(*discv5.Network))
	}

	cfg.Port = 4000
	cfg.UDPPort = 4000

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	s.Start()
	defer s.Stop()

	time.Sleep(2 * time.Second)
	peers := s.host.Network().Peers()
	if len(peers) != 5 {
		t.Errorf("Not all peers added to peerstore, wanted %d but got %d", 5, len(peers))
	}
	// close down all peers
	for _, listener := range listeners {
		listener.Close()
	}
}
