package p2p

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPeer_AtMaxLimit(t *testing.T) {
	// create host and remote peer
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	require.NoError(t, err, "Failed to p2p listen")
	s := &Service{}
	s.peers = peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 0,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: 3,
		},
	})
	s.cfg = &Config{MaxPeers: 0}
	s.addrFilter, err = configureFilter(&Config{})
	require.NoError(t, err)
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	require.NoError(t, err)
	s.host = h1
	defer func() {
		err := h1.Close()
		require.NoError(t, err)
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	require.NoError(t, err, "Failed to p2p listen")
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	require.NoError(t, err)
	defer func() {
		err := h2.Close()
		require.NoError(t, err)
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr, 2000, h1.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	require.NoError(t, err)
	err = h2.Connect(context.Background(), *addrInfo)
	require.NotNil(t, err, "Wanted connection to fail with max peer")
}

func TestPeer_BelowMaxLimit(t *testing.T) {
	// create host and remote peer
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	require.NoError(t, err, "Failed to p2p listen")
	s := &Service{}
	s.peers = peers.NewStatus(context.Background(), &peers.StatusParams{
		PeerLimit: 1,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold: 3,
		},
	})
	s.cfg = &Config{MaxPeers: 1}
	s.addrFilter, err = configureFilter(&Config{})
	require.NoError(t, err)
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	require.NoError(t, err)
	s.host = h1
	defer func() {
		err := h1.Close()
		require.NoError(t, err)
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	require.NoError(t, err, "Failed to p2p listen")
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	require.NoError(t, err)
	defer func() {
		err := h2.Close()
		require.NoError(t, err)
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr, 2000, h1.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	require.NoError(t, err)
	err = h2.Connect(context.Background(), *addrInfo)
	assert.NoError(t, err, "Wanted connection to succeed")
}

func TestPeerAllowList(t *testing.T) {
	// create host with allow list
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	// use unattainable subnet, which will lead to
	// peer rejecting all peers, except for those
	// from that subnet.
	cidr := "202.35.89.12/16"

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	require.NoError(t, err, "Failed to p2p listen")
	s := &Service{}
	s.addrFilter, err = configureFilter(&Config{AllowListCIDR: cidr})
	require.NoError(t, err)
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	require.NoError(t, err)
	s.host = h1
	defer func() {
		err := h1.Close()
		require.NoError(t, err)
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	require.NoError(t, err, "Failed to p2p listen")
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	require.NoError(t, err)
	defer func() {
		err := h2.Close()
		require.NoError(t, err)
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr2, 3000, h2.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	require.NoError(t, err)
	err = h1.Connect(context.Background(), *addrInfo)
	assert.NotNil(t, err, "Wanted connection to fail with allow list")
	assert.ErrorContains(t, "no good addresses", err)
}

func TestPeerDenyList(t *testing.T) {
	// create host with deny list
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	mask := ipAddr2.DefaultMask()
	ones, _ := mask.Size()
	maskedIP := ipAddr2.Mask(mask)
	cidr := maskedIP.String() + fmt.Sprintf("/%d", ones)

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	require.NoError(t, err, "Failed to p2p listen")
	s := &Service{}
	s.addrFilter, err = configureFilter(&Config{DenyListCIDR: []string{cidr}})
	require.NoError(t, err)
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	require.NoError(t, err)
	s.host = h1
	defer func() {
		err := h1.Close()
		require.NoError(t, err)
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	require.NoError(t, err, "Failed to p2p listen")
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	require.NoError(t, err)
	defer func() {
		err := h2.Close()
		require.NoError(t, err)
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr2, 3000, h2.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	require.NoError(t, err)
	err = h1.Connect(context.Background(), *addrInfo)
	assert.NotNil(t, err, "Wanted connection to fail with deny list")
	assert.ErrorContains(t, "no good addresses", err)
}

func TestService_InterceptAddrDial_Allow(t *testing.T) {
	s := &Service{}
	var err error
	cidr := "212.67.89.112/16"
	s.addrFilter, err = configureFilter(&Config{AllowListCIDR: cidr})
	require.NoError(t, err)
	ip := "212.67.10.122"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	require.NoError(t, err)
	valid := s.InterceptAddrDial("", multiAddress)
	if !valid {
		t.Errorf("Expected multiaddress with ip %s to not be rejected with an allow cidr mask of %s", ip, cidr)
	}
}
