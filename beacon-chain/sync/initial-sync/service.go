package initialsync

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var _ = shared.Service(&InitialSync{})

type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.HeadRetriever
	blockchain.FinalizationRetriever
	blockchain.AttestationReceiver
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
	chain blockchainService
	p2p p2p.P2P
}

func NewInitialSync(cfg *Config) *InitialSync {
	return &InitialSync{
		helloTracker: cfg.RegSync,
		chain: cfg.Chain,
		p2p: cfg.P2P,
	}
}

func (s *InitialSync) Start() {
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

	for headSlot := s.chain.HeadSlot(); headSlot < best.HeadSlot; {
		req := &pb.BeaconBlocksRequest{
			HeadSlot: headSlot,
			HeadBlockRoot: s.chain.HeadRoot(),
			Count: 64,
			Step: 1,
		}

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
		if  err := s.p2p.Encoding().Decode(strm, resp); err != nil {
			panic(err)
		}
	}
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
