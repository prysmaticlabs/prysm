package p2p

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
)

func TestPeer_AtMaxLimit(t *testing.T) {
	// create host and remote peer
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	s := &Service{}
	s.peers = peers.NewStatus(3)
	s.cfg = &Config{MaxPeers: 0}
	s.addrFilter, err = configureFilter(&Config{})
	if err != nil {
		t.Fatal(err)
	}
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	if err != nil {
		t.Fatal(err)
	}
	s.host = h1
	defer func() {
		err := h1.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := h2.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr, 2000, h1.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	if err != nil {
		t.Fatal(err)
	}
	err = h2.Connect(context.Background(), *addrInfo)
	if err == nil {
		t.Error("Wanted connection to fail with max peer")
	}
}

func TestPeer_BelowMaxLimit(t *testing.T) {
	// create host and remote peer
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	s := &Service{}
	s.peers = peers.NewStatus(3)
	s.cfg = &Config{MaxPeers: 1}
	s.addrFilter, err = configureFilter(&Config{})
	if err != nil {
		t.Fatal(err)
	}
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	if err != nil {
		t.Fatal(err)
	}
	s.host = h1
	defer func() {
		err := h1.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := h2.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr, 2000, h1.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	if err != nil {
		t.Fatal(err)
	}
	err = h2.Connect(context.Background(), *addrInfo)
	if err != nil {
		t.Errorf("Wanted connection to succeed: %v", err)
	}
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
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	s := &Service{}
	s.addrFilter, err = configureFilter(&Config{AllowListCIDR: cidr})
	if err != nil {
		t.Fatal(err)
	}
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	if err != nil {
		t.Fatal(err)
	}
	s.host = h1
	defer func() {
		err := h1.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := h2.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr2, 3000, h2.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	if err != nil {
		t.Fatal(err)
	}
	err = h1.Connect(context.Background(), *addrInfo)
	if err == nil {
		t.Error("Wanted connection to fail with allow list")
	}
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
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	s := &Service{}
	s.addrFilter, err = configureFilter(&Config{DenyListCIDR: []string{cidr}})
	if err != nil {
		t.Fatal(err)
	}
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), libp2p.ConnectionGater(s)}...)
	if err != nil {
		t.Fatal(err)
	}
	s.host = h1
	defer func() {
		err := h1.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := h2.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr2, 3000, h2.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	if err != nil {
		t.Fatal(err)
	}
	err = h1.Connect(context.Background(), *addrInfo)
	if err == nil {
		t.Error("Wanted connection to fail with deny list")
	}
}

func TestService_InterceptAddrDial_Allow(t *testing.T) {
	s := &Service{}
	var err error
	cidr := "212.67.89.112/16"
	s.addrFilter, err = configureFilter(&Config{AllowListCIDR: cidr})
	if err != nil {
		t.Fatal(err)
	}
	ip := "212.67.10.122"
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 3000))
	if err != nil {
		t.Fatal(err)
	}
	valid := s.InterceptAddrDial("", multiAddress)
	if !valid {
		t.Errorf("Expected multiaddress with ip %s to not be rejected with an allow cidr mask of %s", ip, cidr)
	}
}
