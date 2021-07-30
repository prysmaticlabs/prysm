package p2p

import (
	"context"
	"math/rand"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestStartDiscv5_DifferentForkDigests(t *testing.T) {
	port := 2000
	ipAddr, pkey := createAddrAndPrivKey(t)
	genesisTime := time.Now()
	genesisValidatorsRoot := make([]byte, 32)
	s := &Service{
		cfg: &Config{
			UDPPort:       uint(port),
			StateNotifier: &mock.MockStateNotifier{},
		},
		genesisTime:           genesisTime,
		genesisValidatorsRoot: genesisValidatorsRoot,
	}
	bootListener, err := s.createListener(ipAddr, pkey)
	require.NoError(t, err)
	defer bootListener.Close()

	bootNode := bootListener.Self()
	cfg := &Config{
		Discv5BootStrapAddr: []string{bootNode.String()},
		UDPPort:             uint(port),
		StateNotifier:       &mock.MockStateNotifier{},
	}

	var listeners []*discover.UDPv5
	for i := 1; i <= 5; i++ {
		port = 3000 + i
		cfg.UDPPort = uint(port)
		ipAddr, pkey := createAddrAndPrivKey(t)

		// We give every peer a different genesis validators root, which
		// will cause each peer to have a different ForkDigest, preventing
		// them from connecting according to our discovery rules for Ethereum consensus.
		root := make([]byte, 32)
		copy(root, strconv.Itoa(port))
		s = &Service{
			cfg:                   cfg,
			genesisTime:           genesisTime,
			genesisValidatorsRoot: root,
		}
		listener, err := s.startDiscoveryV5(ipAddr, pkey)
		assert.NoError(t, err, "Could not start discovery for node")
		listeners = append(listeners, listener)
	}
	defer func() {
		// Close down all peers.
		for _, listener := range listeners {
			listener.Close()
		}
	}()

	// Wait for the nodes to have their local routing tables to be populated with the other nodes
	time.Sleep(discoveryWaitTime)

	lastListener := listeners[len(listeners)-1]
	nodes := lastListener.Lookup(bootNode.ID())
	if len(nodes) < 4 {
		t.Errorf("The node's local table doesn't have the expected number of nodes. "+
			"Expected more than or equal to %d but got %d", 4, len(nodes))
	}

	// Now, we start a new p2p service. It should have no peers aside from the
	// bootnode given all nodes provided by discv5 will have different fork digests.
	cfg.UDPPort = 14000
	cfg.TCPPort = 14001
	cfg.MaxPeers = 30
	s, err = NewService(context.Background(), cfg)
	require.NoError(t, err)
	s.genesisTime = genesisTime
	s.genesisValidatorsRoot = make([]byte, 32)
	s.dv5Listener = lastListener
	var addrs []ma.Multiaddr

	for _, n := range nodes {
		if s.filterPeer(n) {
			addr, err := convertToSingleMultiAddr(n)
			require.NoError(t, err)
			addrs = append(addrs, addr)
		}
	}

	// We should not have valid peers if the fork digest mismatched.
	assert.Equal(t, 0, len(addrs), "Expected 0 valid peers")
	require.NoError(t, s.Stop())
}

func TestStartDiscv5_SameForkDigests_DifferentNextForkData(t *testing.T) {
	hook := logTest.NewGlobal()
	logrus.SetLevel(logrus.TraceLevel)
	port := 2000
	ipAddr, pkey := createAddrAndPrivKey(t)
	genesisTime := time.Now()
	genesisValidatorsRoot := make([]byte, 32)
	s := &Service{
		cfg:                   &Config{UDPPort: uint(port)},
		genesisTime:           genesisTime,
		genesisValidatorsRoot: genesisValidatorsRoot,
		stateNotifier:         &mock.MockStateNotifier{},
	}
	bootListener, err := s.createListener(ipAddr, pkey)
	require.NoError(t, err)
	defer bootListener.Close()

	bootNode := bootListener.Self()
	cfg := &Config{
		Discv5BootStrapAddr: []string{bootNode.String()},
		UDPPort:             uint(port),
	}

	params.SetupTestConfigCleanup(t)
	var listeners []*discover.UDPv5
	for i := 1; i <= 5; i++ {
		port = 3000 + i
		cfg.UDPPort = uint(port)
		ipAddr, pkey := createAddrAndPrivKey(t)

		c := params.BeaconConfig()
		nextForkEpoch := types.Epoch(i)
		c.NextForkEpoch = nextForkEpoch
		params.OverrideBeaconConfig(c)

		// We give every peer a different genesis validators root, which
		// will cause each peer to have a different ForkDigest, preventing
		// them from connecting according to our discovery rules for Ethereum consensus.
		s = &Service{
			cfg:                   cfg,
			genesisTime:           genesisTime,
			genesisValidatorsRoot: genesisValidatorsRoot,
			stateNotifier:         &mock.MockStateNotifier{},
		}
		listener, err := s.startDiscoveryV5(ipAddr, pkey)
		assert.NoError(t, err, "Could not start discovery for node")
		listeners = append(listeners, listener)
	}
	defer func() {
		// Close down all peers.
		for _, listener := range listeners {
			listener.Close()
		}
	}()

	// Wait for the nodes to have their local routing tables to be populated with the other nodes
	time.Sleep(discoveryWaitTime)

	lastListener := listeners[len(listeners)-1]
	nodes := lastListener.Lookup(bootNode.ID())
	if len(nodes) < 4 {
		t.Errorf("The node's local table doesn't have the expected number of nodes. "+
			"Expected more than or equal to %d but got %d", 4, len(nodes))
	}

	// Now, we start a new p2p service. It should have no peers aside from the
	// bootnode given all nodes provided by discv5 will have different fork digests.
	cfg.UDPPort = 14000
	cfg.TCPPort = 14001
	cfg.MaxPeers = 30
	cfg.StateNotifier = &mock.MockStateNotifier{}
	s, err = NewService(context.Background(), cfg)
	require.NoError(t, err)

	s.genesisTime = genesisTime
	s.genesisValidatorsRoot = make([]byte, 32)
	s.dv5Listener = lastListener
	var addrs []ma.Multiaddr

	for _, n := range nodes {
		if s.filterPeer(n) {
			addr, err := convertToSingleMultiAddr(n)
			require.NoError(t, err)
			addrs = append(addrs, addr)
		}
	}
	if len(addrs) == 0 {
		t.Error("Expected to have valid peers, got 0")
	}

	require.LogsContain(t, hook, "Peer matches fork digest but has different next fork epoch")
	require.NoError(t, s.Stop())
}

func TestDiscv5_AddRetrieveForkEntryENR(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig()
	c.ForkVersionSchedule = map[types.Epoch][]byte{
		0: params.BeaconConfig().GenesisForkVersion,
		1: {0, 0, 0, 1},
	}
	nextForkEpoch := types.Epoch(1)
	nextForkVersion := []byte{0, 0, 0, 1}
	c.NextForkEpoch = nextForkEpoch
	c.NextForkVersion = nextForkVersion
	params.OverrideBeaconConfig(c)

	genesisTime := time.Now()
	genesisValidatorsRoot := make([]byte, 32)
	digest, err := p2putils.CreateForkDigest(genesisTime, make([]byte, 32))
	require.NoError(t, err)
	enrForkID := &pb.ENRForkID{
		CurrentForkDigest: digest[:],
		NextForkVersion:   nextForkVersion,
		NextForkEpoch:     nextForkEpoch,
	}
	enc, err := enrForkID.MarshalSSZ()
	require.NoError(t, err)
	entry := enr.WithEntry(eth2ENRKey, enc)
	// In epoch 1 of current time, the fork version should be
	// {0, 0, 0, 1} according to the configuration override above.
	temp := t.TempDir()
	randNum := rand.Int()
	tempPath := path.Join(temp, strconv.Itoa(randNum))
	require.NoError(t, os.Mkdir(tempPath, 0700))
	pkey, err := privKey(&Config{DataDir: tempPath})
	require.NoError(t, err, "Could not get private key")
	db, err := enode.OpenDB("")
	require.NoError(t, err)
	localNode := enode.NewLocalNode(db, pkey)
	localNode.Set(entry)

	want, err := helpers.ComputeForkDigest([]byte{0, 0, 0, 0}, genesisValidatorsRoot)
	require.NoError(t, err)

	resp, err := forkEntry(localNode.Node().Record())
	require.NoError(t, err)
	assert.DeepEqual(t, want[:], resp.CurrentForkDigest)
	assert.DeepEqual(t, nextForkVersion, resp.NextForkVersion)
	assert.Equal(t, nextForkEpoch, resp.NextForkEpoch, "Unexpected next fork epoch")
}

func TestAddForkEntry_Genesis(t *testing.T) {
	temp := t.TempDir()
	randNum := rand.Int()
	tempPath := path.Join(temp, strconv.Itoa(randNum))
	require.NoError(t, os.Mkdir(tempPath, 0700))
	pkey, err := privKey(&Config{DataDir: tempPath})
	require.NoError(t, err, "Could not get private key")
	db, err := enode.OpenDB("")
	require.NoError(t, err)

	localNode := enode.NewLocalNode(db, pkey)
	localNode, err = addForkEntry(localNode, time.Now().Add(10*time.Second), bytesutil.PadTo([]byte{'A', 'B', 'C', 'D'}, 32))
	require.NoError(t, err)
	forkEntry, err := forkEntry(localNode.Node().Record())
	require.NoError(t, err)
	assert.DeepEqual(t,
		params.BeaconConfig().GenesisForkVersion, forkEntry.NextForkVersion,
		"Wanted Next Fork Version to be equal to genesis fork version")
}
