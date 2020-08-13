package p2p

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStartDiscV5_DiscoverPeersWithSubnets(t *testing.T) {
	port := 2000
	ipAddr, pkey := createAddrAndPrivKey(t)
	dir, err := testutil.RandDir()
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		assert.NoError(t, err)
	})
	genesisTime := time.Now()
	genesisValidatorsRoot := make([]byte, 32)
	s := &Service{
		cfg:                   &Config{UDPPort: uint(port), DataDir: dir},
		genesisTime:           genesisTime,
		genesisValidatorsRoot: genesisValidatorsRoot,
	}
	bootListener, err := s.createListener(ipAddr, pkey)
	require.NoError(t, err)
	defer bootListener.Close()

	bootNode := bootListener.Self()
	// Use shorter period for testing.
	currentPeriod := pollingPeriod
	pollingPeriod = 1 * time.Second
	defer func() {
		pollingPeriod = currentPeriod
	}()

	var listeners []*discover.UDPv5
	for i := 1; i <= 3; i++ {
		port = 3000 + i
		cfg := &Config{
			BootstrapNodeAddr:   []string{bootNode.String()},
			Discv5BootStrapAddr: []string{bootNode.String()},
			MaxPeers:            30,
			UDPPort:             uint(port),
			DataDir:             dir,
		}
		ipAddr, pkey := createAddrAndPrivKey(t)
		s = &Service{
			cfg:                   cfg,
			genesisTime:           genesisTime,
			genesisValidatorsRoot: genesisValidatorsRoot,
		}
		listener, err := s.startDiscoveryV5(ipAddr, pkey)
		assert.NoError(t, err, "Could not start discovery for node")
		bitV := bitfield.NewBitvector64()
		bitV.SetBitAt(uint64(i), true)

		entry := enr.WithEntry(attSubnetEnrKey, &bitV)
		listener.LocalNode().Set(entry)
		listeners = append(listeners, listener)
	}
	defer func() {
		// Close down all peers.
		for _, listener := range listeners {
			listener.Close()
		}
	}()

	// Make one service on port 3001.
	port = 4000
	cfg := &Config{
		BootstrapNodeAddr:   []string{bootNode.String()},
		Discv5BootStrapAddr: []string{bootNode.String()},
		MaxPeers:            30,
		UDPPort:             uint(port),
		DataDir:             dir,
	}
	cfg.StateNotifier = &mock.MockStateNotifier{}
	s, err = NewService(cfg)
	require.NoError(t, err)
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

	// Wait for the nodes to have their local routing tables to be populated with the other nodes
	time.Sleep(6 * discoveryWaitTime)

	// look up 3 different subnets
	ctx := context.Background()
	exists, err := s.FindPeersWithSubnet(ctx, 1)
	require.NoError(t, err)
	exists2, err := s.FindPeersWithSubnet(ctx, 2)
	require.NoError(t, err)
	exists3, err := s.FindPeersWithSubnet(ctx, 3)
	require.NoError(t, err)
	if !exists || !exists2 || !exists3 {
		t.Fatal("Peer with subnet doesn't exist")
	}

	// Update ENR of a peer.
	testService := &Service{
		dv5Listener: listeners[0],
		metaData:    &pb.MetaData{},
		cfg:         cfg,
	}
	cache.SubnetIDs.AddAttesterSubnetID(0, 10)
	testService.RefreshENR()
	time.Sleep(2 * time.Second)

	exists, err = s.FindPeersWithSubnet(ctx, 2)
	require.NoError(t, err)

	assert.Equal(t, true, exists, "Peer with subnet doesn't exist")
	assert.NoError(t, s.Stop())
	exitRoutine <- true
}
