package initialsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

var _ = shared.Service(&InitialSync{})

type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.HeadFetcher
	blockchain.ChainFeeds
}

const (
	minHelloCount            = 1               // TODO(3147): Set this to more than 1, maybe configure from flag?
	handshakePollingInterval = 5 * time.Second // Polling interval for checking the number of received handshakes.
)

// Config to set up the initial sync service.
type Config struct {
	P2P     p2p.P2P
	DB      db.Database
	Chain   blockchainService
	RegSync sync.HelloTracker
}

// InitialSync service.
type InitialSync struct {
	helloTracker sync.HelloTracker
	chain        blockchainService
	p2p          p2p.P2P
	synced       bool
	chainStarted bool
}

// NewInitialSync configures the initial sync service responsible for bringing the node up to the
// latest head of the blockchain.
func NewInitialSync(cfg *Config) *InitialSync {
	return &InitialSync{
		helloTracker: cfg.RegSync,
		chain:        cfg.Chain,
		p2p:          cfg.P2P,
	}
}

// Start the initial sync service.
func (s *InitialSync) Start() {
	ch := make(chan time.Time)
	sub := s.chain.StateInitializedFeed().Subscribe(ch)
	defer sub.Unsubscribe()

	// Wait until chain start.
	genesis := <-ch
	if genesis.After(roughtime.Now()) {
		time.Sleep(roughtime.Until(genesis))
	}
	s.chainStarted = true
	currentSlot := slotsSinceGenesis(genesis)
	if helpers.SlotToEpoch(currentSlot) == 0 {
		log.Info("Chain started within the last epoch. Not syncing.")
		s.synced = true
		return
	}

	// Are we already in sync, or close to it?
	if helpers.SlotToEpoch(s.chain.HeadSlot()) == helpers.SlotToEpoch(currentSlot) {
		log.Info("Already synced to the current epoch.")
		s.synced = true
		return
	}

	// Every 5 sec, report handshake count.
	for {
		helloCount := len(s.helloTracker.Hellos())
		log.WithField(
			"hellos",
			fmt.Sprintf("%d/%d", helloCount, minHelloCount),
		).Info("Waiting for enough peer handshakes before syncing.")

		if helloCount >= minHelloCount {
			break
		}
		time.Sleep(handshakePollingInterval)
	}

	pid, best := bestHello(s.helloTracker.Hellos())

	var last *eth.BeaconBlock
	for headSlot := s.chain.HeadSlot(); headSlot < slotsSinceGenesis(genesis); {
		req := &pb.BeaconBlocksRequest{
			HeadSlot:      headSlot + 1,
			HeadBlockRoot: s.chain.HeadRoot(),
			Count:         64,
			Step:          1,
		}

		log.WithField("data", fmt.Sprintf("%+v", req)).Info("Sending msg")

		strm, err := s.p2p.Send(context.Background(), req, pid)
		if err != nil {
			panic(err)
		}

		// Read status code.
		code, errMsg, err := sync.ReadStatusCode(strm, s.p2p.Encoding())
		if err != nil {
			panic(err)
		}
		if code != 0 {
			log.Errorf("Request failed. Request was %+v", req)
			panic(errMsg)
		}

		resp := make([]*eth.BeaconBlock, 0)
		if err := s.p2p.Encoding().DecodeWithLength(strm, &resp); err != nil {
			log.Error(err)
			continue
		}

		for _, blk := range resp {
			if blk.Slot <= headSlot {
				continue
			}
			if blk.Slot < helpers.StartSlot(best.FinalizedEpoch+1) {
				if err := s.chain.ReceiveBlockNoPubsubForkchoice(context.Background(), blk); err != nil {
					panic(err)
				}
			} else {
				if err := s.chain.ReceiveBlockNoPubsub(context.Background(), blk); err != nil {
					panic(err)
				}
			}
			last = blk
		}

		headSlot = s.chain.HeadSlot()
	}

	// Force a fork choice update since fork choice was not run during initial sync.
	if err := s.chain.ReceiveBlockNoPubsub(context.Background(), last); err != nil {
		panic(err)
	}

	log.Infof("Synced up to %d", s.chain.HeadSlot())
	s.synced = true
}

// Stop initial sync.
func (s *InitialSync) Stop() error {
	return nil
}

// Status of initial sync.
func (s *InitialSync) Status() error {
	if !s.synced && s.chainStarted {
		return errors.New("syncing")
	}
	return nil
}

func bestHello(data map[peer.ID]*pb.Hello) (peer.ID, *pb.Hello) {
	for pid, hello := range data {
		return pid, hello
	}

	return "", nil
}

func slotsSinceGenesis(genesisTime time.Time) uint64 {
	return uint64(roughtime.Since(genesisTime).Seconds()) / params.BeaconConfig().SecondsPerSlot
}
