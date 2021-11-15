package fuzz

import (
	"context"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	regularsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

var p *p2p.Service
var h host.Host

func init() {
	logrus.SetLevel(logrus.PanicLevel)

	var err error
	p, err = p2p.NewService(context.Background(), p2p.WithNoDiscovery())
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
	regularsync.NewService(context.Background(),
		regularsync.WithP2P(p),
		regularsync.WithChainService(
			&mock.ChainService{
				Root:                bytesutil.PadTo([]byte("root"), 32),
				FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 4, Root: make([]byte, 32)},
				Fork:                &ethpb.Fork{CurrentVersion: []byte("foo")},
			}),
		regularsync.WithStateNotifier((&mock.ChainService{}).StateNotifier()),
		regularsync.WithOperationNotifier((&mock.ChainService{}).OperationNotifier()),
		regularsync.WithInitialSync(&mockSync.Sync{IsSyncing: false}),
	)
}

// FuzzP2PRPCStatus wraps BeaconFuzzP2PRPCStatus in a go-fuzz compatible interface
func FuzzP2PRPCStatus(b []byte) int {
	BeaconFuzzP2PRPCStatus(b)
	return 0
}

// BeaconFuzzP2PRPCStatus implements libfuzzer and beacon fuzz interface.
func BeaconFuzzP2PRPCStatus(b []byte) {
	s, err := h.NewStream(context.Background(), p.PeerID(), "/eth2/beacon_chain/req/status/1/ssz_snappy")
	if err != nil {
		// libp2p ¯\_(ツ)_/¯
		if strings.Contains(err.Error(), "stream reset") || strings.Contains(err.Error(), "connection reset by peer") || strings.Contains(err.Error(), "max dial attempts exceeded") {
			return
		}
		panic(errors.Wrap(err, "failed to open stream"))
	}
	if s == nil {
		panic("nil stream")
	}
	defer func() {
		err := s.Close()
		_ = err
	}()
	_, err = s.Write(b)
	_ = err
}
