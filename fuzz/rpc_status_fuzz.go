package fuzz

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Set up servers
var svc *sync.Service
var p *p2p.Service
var h host.Host

func init() {
	var err error
	p, err = p2p.NewService(&p2p.Config{
		NoDiscovery:           true,
		StaticPeers:           nil,
		BootstrapNodeAddr:     nil,
		KademliaBootStrapAddr: nil,
		Discv5BootStrapAddr:   nil,
		RelayNodeAddr:         "",
		LocalIP:               "",
		HostAddress:           "",
		HostDNS:               "",
		PrivateKey:            "",
		DataDir:               "",
		TCPPort:               0,
		UDPPort:               0,
		MaxPeers:              0,
		WhitelistCIDR:         "",
		EnableUPnP:            false,
		EnableDiscv5:          false,
		Encoding:              "ssz",
	})
	if err != nil {
		panic(errors.Wrap(err, "could not create new p2p service"))
	}

	h, err = libp2p.New(context.Background())
	if err != nil {
		panic(errors.Wrap(err, "could not create new libp2p host"))
	}

	info := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
	if err := p.Connect(info); err != nil {
		panic(errors.Wrap(err, "could not connect to peer"))
	}
	svc = sync.NewRegularSync(&sync.Config{
		P2P:                 p,
		DB:                  nil,
		AttPool:             nil,
		ExitPool:            nil,
		SlashingPool:        nil,
		Chain:               nil,
		StateNotifier:       (&mock.ChainService{}).StateNotifier(),
		AttestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		InitialSync:         &mockSync.Sync{IsSyncing: false},
		StateSummaryCache:   cache.NewStateSummaryCache(),
		BlockNotifier:       nil,
	})
}

func FuzzP2PRPCStatus(b []byte) {
	if len(b) < (&pb.Status{}).SizeSSZ() || len(b) > (&pb.Status{}).SizeSSZ()+10 {
		return
	}
	s, err := h.NewStream(context.Background(), p.PeerID(), "/eth2/beacon_chain/req/status/1/ssz")
	if err != nil {
		panic(errors.Wrap(err, "could not open stream"))
	}
	defer s.Conn()
	if s == nil {
		panic("nil stream")
	}
	if _, err := s.Write(b); err != nil {
		panic(errors.Wrap(err, "could not write to stream"))
	}
}
