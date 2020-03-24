package p2p

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
)

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

	// update ENR of a peer.
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
