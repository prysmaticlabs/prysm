package fuzz

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var s network.Stream

func init() {
	logrus.SetLevel(logrus.PanicLevel)

	p, err := p2p.NewService(&p2p.Config{
		NoDiscovery: true,
		Encoding:    "ssz",
	})
	if err != nil {
		panic(errors.Wrap(err, "could not create new p2p service"))
	}

	h, err := libp2p.New(context.Background())
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
	sync.NewRegularSync(&sync.Config{
		P2P:          p,
		DB:           nil,
		AttPool:      nil,
		ExitPool:     nil,
		SlashingPool: nil,
		Chain: &mock.ChainService{
			Root:                []byte("root"),
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 4},
			Fork:                &pb.Fork{CurrentVersion: []byte("foo")},
		},
		StateNotifier:       (&mock.ChainService{}).StateNotifier(),
		AttestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		InitialSync:         &mockSync.Sync{IsSyncing: false},
		StateSummaryCache:   cache.NewStateSummaryCache(),
		BlockNotifier:       nil,
	})

	s, err = h.NewStream(context.Background(), p.PeerID(), "/eth2/beacon_chain/req/status/1/ssz")
	if err != nil {
		panic(errors.Wrap(err, "could not open stream"))
	}
	if s == nil {
		panic("nil stream")
	}
}

func FuzzP2PRPCStatus(b []byte) {
	if _, err := s.Write(b); err != nil {
		panic(err)
	}
}
