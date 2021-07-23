package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v2/wrapper"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStartDiscV5_DiscoverPeersWithSubnets(t *testing.T) {
	// This test needs to be entirely rewritten and should be done in a follow up PR from #7885.
	t.Skip("This test is now failing after PR 7885 due to false positive")
	port := 2000
	ipAddr, pkey := createAddrAndPrivKey(t)
	genesisTime := time.Now()
	genesisValidatorsRoot := make([]byte, 32)
	s := &Service{
		cfg:                   &Config{UDPPort: uint(port)},
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

	// Make one service on port 4001.
	port = 4001
	cfg := &Config{
		BootstrapNodeAddr:   []string{bootNode.String()},
		Discv5BootStrapAddr: []string{bootNode.String()},
		MaxPeers:            30,
		UDPPort:             uint(port),
	}
	cfg.StateNotifier = &mock.MockStateNotifier{}
	s, err = NewService(context.Background(), cfg)
	require.NoError(t, err)
	exitRoutine := make(chan bool)
	go func() {
		s.Start()
		<-exitRoutine
	}()
	time.Sleep(50 * time.Millisecond)
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
	exists, err := s.FindPeersWithSubnet(ctx, "", 1, params.BeaconNetworkConfig().MinimumPeersInSubnet)
	require.NoError(t, err)
	exists2, err := s.FindPeersWithSubnet(ctx, "", 2, params.BeaconNetworkConfig().MinimumPeersInSubnet)
	require.NoError(t, err)
	exists3, err := s.FindPeersWithSubnet(ctx, "", 3, params.BeaconNetworkConfig().MinimumPeersInSubnet)
	require.NoError(t, err)
	if !exists || !exists2 || !exists3 {
		t.Fatal("Peer with subnet doesn't exist")
	}

	// Update ENR of a peer.
	testService := &Service{
		dv5Listener: listeners[0],
		metaData: wrapper.WrappedMetadataV0(&pb.MetaDataV0{
			Attnets: bitfield.NewBitvector64(),
		}),
	}
	cache.SubnetIDs.AddAttesterSubnetID(0, 10)
	testService.RefreshENR()
	time.Sleep(2 * time.Second)

	exists, err = s.FindPeersWithSubnet(ctx, "", 2, params.BeaconNetworkConfig().MinimumPeersInSubnet)
	require.NoError(t, err)

	assert.Equal(t, true, exists, "Peer with subnet doesn't exist")
	assert.NoError(t, s.Stop())
	exitRoutine <- true
}
