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
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
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

	if port != listener.Self().UDP() {
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
		Discv5BootStrapAddr: []string{bootNode.String()},
		Encoding:            "ssz",
	}

	var listeners []*discover.UDPv5
	for i := 1; i <= 1; i++ {
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

func TestStartDiscV5_DiscoverPeersWithSubnets(t *testing.T) {
	port := 2000
	ipAddr, pkey := createAddrAndPrivKey(t)
	bootListener := createListener(ipAddr, pkey, &Config{UDPPort: uint(port)})
	defer bootListener.Close()

	bootNode := bootListener.Self()
	cfg := &Config{
		BootstrapNodeAddr:   []string{bootNode.String()},
		Discv5BootStrapAddr: []string{bootNode.String()},
		Encoding:            "ssz",
		MaxPeers:            30,
	}
	// Use shorter period for testing.
	currentPeriod := pollingPeriod
	pollingPeriod = 1 * time.Second
	defer func() {
		pollingPeriod = currentPeriod
	}()

	var listeners []*discover.UDPv5
	for i := 1; i <= 3; i++ {
		port = 3000 + i
		cfg.UDPPort = uint(port)
		ipAddr, pkey := createAddrAndPrivKey(t)
		listener, err := startDiscoveryV5(ipAddr, pkey, cfg)
		if err != nil {
			t.Errorf("Could not start discovery for node: %v", err)
		}
		bitV := bitfield.NewBitvector64()
		bitV.SetBitAt(uint64(i), true)

		entry := enr.WithEntry(attSubnetEnrKey, &bitV)
		listener.LocalNode().Set(entry)
		listeners = append(listeners, listener)
	}

	// Make one service on port 3001.
	port = 4000
	cfg.UDPPort = uint(port)
	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	defer s.Stop()

	// Wait for the nodes to have their local routing tables to be populated with the other nodes
	time.Sleep(discoveryWaitTime)

	// look up 3 different subnets
	exists, err := s.FindPeersWithSubnet(1)
	if err != nil {
		t.Fatal(err)
	}
	exists2, err := s.FindPeersWithSubnet(2)
	if err != nil {
		t.Fatal(err)
	}
	exists3, err := s.FindPeersWithSubnet(3)
	if err != nil {
		t.Fatal(err)
	}
	if !exists || !exists2 || !exists3 {
		t.Fatal("Peer with subnet doesn't exist")
	}

	// update ENR of a peer
	testService := &Service{dv5Listener: listeners[0]}
	cache.CommitteeIDs.AddIDs([]uint64{10}, 0)
	testService.RefreshENR(0)
	time.Sleep(2 * time.Second)

	exists, err = s.FindPeersWithSubnet(2)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("Peer with subnet doesn't exist")
	}

}

func TestMultiAddrsConversion_InvalidIPAddr(t *testing.T) {
	addr := net.ParseIP("invalidIP")
	_, pkey := createAddrAndPrivKey(t)
	node, err := createLocalNode(pkey, addr, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	multiAddr := convertToMultiAddr([]*enode.Node{node.Node()})
	if len(multiAddr) != 0 {
		t.Error("Invalid ip address converted successfully")
	}
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
	cfg := &Config{Encoding: "ssz", MaxPeers: 30}
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
	time.Sleep(100 * time.Millisecond)

	peers := s.host.Network().Peers()
	if len(peers) != 5 {
		t.Errorf("Not all peers added to peerstore, wanted %d but got %d", 5, len(peers))
	}
}
