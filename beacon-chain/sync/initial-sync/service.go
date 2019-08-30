package initialsync

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/go-ssz"
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
	blockchain.HeadRetriever
	blockchain.FinalizationRetriever
	blockchain.AttestationReceiver
	blockchain.ChainFeeds
}

const minHelloCount = 1 // TODO: Set this to more than 1, maybe configure from flag?

// Config to set up the initial sync service.
type Config struct {
	P2P     p2p.P2P
	DB      db.Database
	Chain   blockchainService
	RegSync sync.HelloTracker
}

type InitialSync struct {
	helloTracker sync.HelloTracker
	chain        blockchainService
	p2p          p2p.P2P
	db           db.Database
}

func NewInitialSync(cfg *Config) *InitialSync {
	return &InitialSync{
		helloTracker: cfg.RegSync,
		chain:        cfg.Chain,
		p2p:          cfg.P2P,
		db:           cfg.DB,
	}
}

func (s *InitialSync) Start() {
	ch := make(chan time.Time)
	sub :=  s.chain.StateInitializedFeed().Subscribe(ch)
	defer sub.Unsubscribe()

	// Wait until chain start.
	genesis := <-ch
	if genesis.After(roughtime.Now()) {
		time.Sleep(roughtime.Until(genesis))
	}
	currentSlot := uint64(roughtime.Since(genesis).Seconds()) / params.BeaconConfig().SecondsPerSlot
	if helpers.SlotToEpoch(currentSlot) == 0 {
		log.Info("Chain started within the last epoch. Not syncing.")
		return
	}

	// Get last saved head block in DB
	headBlock, _ := s.db.HeadBlock(context.Background())
	headRoot, _ := ssz.SigningRoot(headBlock)

	// Are we already in sync, or close to it?
	if helpers.SlotToEpoch(headBlock.Slot) == helpers.SlotToEpoch(currentSlot) {
		log.Info("Already synced to the current epoch.")
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

		time.Sleep(5 * time.Second)
	}

	pid, best := bestHello(s.helloTracker.Hellos())

	var last *eth.BeaconBlock
	for headSlot := headBlock.Slot; headSlot < uint64(roughtime.Since(genesis).Seconds()) / params.BeaconConfig().SecondsPerSlot; {
		req := &pb.BeaconBlocksRequest{
			HeadSlot:      headSlot,
			HeadBlockRoot: headRoot[:],
			Count:         64,
			Step:          1,
		}

		log.WithField("data", fmt.Sprintf("%+v", req)).Info("Sending msg")

		strm, err := s.p2p.Send(context.Background(), req, pid)
		if err != nil {
			panic(err)
		}

		// Read status code
		code, errMsg, err := sync.ReadStatusCode(strm, s.p2p.Encoding())
		if err != nil {
			panic(err)
		}
		if code != 0 {
			log.Errorf("Request failed. Request was %+v", req)
			panic(errMsg.ErrorMessage)
		}

		resp := &pb.BeaconBlocksResponse{}
		if err := s.p2p.Encoding().Decode(strm, resp); err != nil {
			panic(err)
		}

		log.Infof("Received %d blocks", len(resp.Blocks))

		for _, blk := range resp.Blocks {
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
}

func bestHello(data map[peer.ID]*pb.Hello) (peer.ID, *pb.Hello) {
	for pid, hello := range data {
		return pid, hello
	}

	return "", nil
}

func (s *InitialSync) Stop() error {
	return nil
}

func (s *InitialSync) Status() error {
	return nil
}
