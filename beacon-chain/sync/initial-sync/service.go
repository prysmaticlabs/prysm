package initialsync

import (
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&InitialSync{})

type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.HeadRetriever
	blockchain.FinalizationRetriever
	blockchain.AttestationReceiver
}

const minHelloCount = 3

// Config to set up the initial sync service.
type Config struct {
	P2P     p2p.P2P
	DB      db.Database
	Chain   blockchainService
	RegSync sync.HelloTracker
}

type InitialSync struct {
	helloTracker sync.HelloTracker
}

func NewInitialSync(cfg *Config) *InitialSync {
	return &InitialSync{
		helloTracker: cfg.RegSync,
	}
}

func (s *InitialSync) Start() {
	// Every 5 sec, report handshake count.
	for {
		helloCount := len(s.helloTracker.Hellos())
		if helloCount > minHelloCount {
			break
		}


		log.WithField(
			"hellos received",
			fmt.Sprintf("%d/%d", helloCount, minHelloCount),
		).Info("Waiting for enough peer handshakes before syncing.")

		time.Sleep(5 * time.Second)
	}
}

func (s *InitialSync) Stop() error {
	return nil
}

func (s *InitialSync) Status() error {
	return nil
}
