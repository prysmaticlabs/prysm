package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
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

func newhost(t *testing.T, options []libp2p.Option) *basichost.BasicHost {
	ctx := context.Background()
	h := basichost.New(swarmt.GenSwarm(t, ctx), options)
	_, err := pubsub.NewFloodSub(ctx, h,
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	return h
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
	ipAddr, pkey := createAddrAndPrivKey(t)
	bootListener := createListener(ipAddr, port, pkey)
	defer bootListener.Close()

	bootNode := bootListener.Self()

	cfg := &Config{
		BootstrapNodeAddr: bootNode.String(),
	}

	// setup other nodes
	var listeners []*discv5.Network
	for i := 1; i <= 5; i++ {
		port = 2000 + i
		cfg.UDPPort = uint(port)
		cfg.Port = uint(port)
		ipAddr, pkey := createAddrAndPrivKey(t)
		opts, ipAddr, pkey := buildOptions(cfg)
		_ = newhost(t, opts)
		listener, err := startDiscoveryV5(ipAddr, pkey, cfg)
		if err != nil {
			t.Errorf("Could not start discovery for node: %v", err)
		}
		listeners = append(listeners, listener)
	}
	cfg.UDPPort = 4000
	s, _ := NewService(cfg)

	defer s.Stop()
	s.Start()

	time.Sleep(5 * time.Second)

	peers := s.host.Network().Peers()

	if len(peers) != 12 {
		t.Errorf("Not all peers added to peerstore, wanted %d but got %d", 12, len(peers))
	}
}
