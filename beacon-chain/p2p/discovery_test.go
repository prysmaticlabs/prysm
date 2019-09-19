package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var discoveryWaitTime = 1 * time.Second

func init() {
	rand.Seed(time.Now().Unix())
}

func createAddrAndPrivKey(t *testing.T) (net.IP, *ecdsa.PrivateKey) {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		t.Fatalf("Could not get ip: %v", err)
	}
	ipAddr := net.ParseIP(ip)
	temp := testutil.TempDir()
	randNum := rand.Int()
	tempPath := path.Join(temp, strconv.Itoa(randNum))
	err = os.Mkdir(tempPath, 0700)
	if err != nil {
		t.Fatal(err)
	}
	pkey, err := privKey(&Config{Encoding: "ssz", DataDir: tempPath})
	if err != nil {
		t.Fatalf("Could not get private key: %v", err)
	}
	return ipAddr, pkey
}

func TestCreateListener(t *testing.T) {
	port := 1024
	ipAddr, pkey := createAddrAndPrivKey(t)
	listener := createListener(ipAddr, pkey, &Config{UDPPort: uint(port)})
	defer listener.Close()

	if !listener.Self().IP().Equal(ipAddr) {
		t.Errorf("Ip address is not the expected type, wanted %s but got %s", ipAddr.String(), listener.Self().IP().String())
	}

	if port != int(listener.Self().UDP()) {
		t.Errorf("In correct port number, wanted %d but got %d", port, listener.Self().UDP())
	}
	pubkey := listener.Self().Pubkey()
	XisSame := pkey.PublicKey.X.Cmp(pubkey.X) == 0
	YisSame := pkey.PublicKey.Y.Cmp(pubkey.Y) == 0

	if !(XisSame && YisSame) {
		t.Error("Pubkey is different from what was used to create the listener")
	}
}

func TestStartDiscV5_DiscoverAllPeers(t *testing.T) {
	port := 2000
	ipAddr, pkey := createAddrAndPrivKey(t)
	bootListener := createListener(ipAddr, pkey, &Config{UDPPort: uint(port)})
	defer bootListener.Close()

	bootNode := bootListener.Self()
	cfg := &Config{
		BootstrapNodeAddr: bootNode.String(),
		Encoding:          "ssz",
	}

	var listeners []*discover.UDPv5
	for i := 1; i <= 5; i++ {
		port = 3000 + i
		cfg.UDPPort = uint(port)
		ipAddr, pkey := createAddrAndPrivKey(t)
		listener, err := startDiscoveryV5(ipAddr, pkey, cfg)
		if err != nil {
			t.Errorf("Could not start discovery for node: %v", err)
		}
		listeners = append(listeners, listener)
	}

	// Wait for the nodes to have their local routing tables to be populated with the other nodes
	time.Sleep(discoveryWaitTime)

	lastListener := listeners[len(listeners)-1]
	nodes := lastListener.Lookup(bootNode.ID())
	if len(nodes) < 4 {
		t.Errorf("The node's local table doesn't have the expected number of nodes. "+
			"Expected more than or equal to %d but got %d", 4, len(nodes))
	}

	// Close all ports
	for _, listener := range listeners {
		listener.Close()
	}
}

func TestMultiAddrsConversion_InvalidIPAddr(t *testing.T) {
	hook := logTest.NewGlobal()
	ipAddr := net.IPv6zero
	_, pkey := createAddrAndPrivKey(t)
	node, err := createLocalNode(pkey, ipAddr, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = convertToMultiAddr([]*enode.Node{node.Node()})
	testutil.AssertLogsContain(t, hook, "node doesn't have an ip4 address")
}

func TestMultiAddrConversion_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	ipAddr, pkey := createAddrAndPrivKey(t)
	listener := createListener(ipAddr, pkey, &Config{})

	_ = convertToMultiAddr([]*enode.Node{listener.Self()})
	testutil.AssertLogsDoNotContain(t, hook, "Node doesn't have an ip4 address")
	testutil.AssertLogsDoNotContain(t, hook, "Invalid port, the tcp port of the node is a reserved port")
	testutil.AssertLogsDoNotContain(t, hook, "Could not get multiaddr")
}

func TestStaticPeering_PeersAreAdded(t *testing.T) {
	cfg := &Config{Encoding: "ssz"}
	port := 3000
	var staticPeers []string
	var hosts []host.Host
	// setup other nodes
	for i := 1; i <= 5; i++ {
		h, _, ipaddr := createHost(t, port+i)
		staticPeers = append(staticPeers, fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipaddr, port+i, h.ID()))
		hosts = append(hosts, h)
	}

	defer func() {
		for _, h := range hosts {
			_ = h.Close()
		}
	}()

	cfg.TCPPort = 14001
	cfg.UDPPort = 14000
	cfg.StaticPeers = staticPeers

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	s.Start()
	s.dv5Listener = &mockListener{}
	defer s.Stop()

	peers := s.host.Network().Peers()
	if len(peers) != 5 {
		t.Errorf("Not all peers added to peerstore, wanted %d but got %d", 5, len(peers))
	}
}
