package initialsync

import (
	"errors"
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
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
	minStatusCount           = 1               // TODO(3147): Set this to more than 1, maybe configure from flag?
	handshakePollingInterval = 5 * time.Second // Polling interval for checking the number of received handshakes.
)

// Config to set up the initial sync service.
type Config struct {
	P2P   p2p.P2P
	DB    db.Database
	Chain blockchainService
}

// InitialSync service.
type InitialSync struct {
	chain        blockchainService
	p2p          p2p.P2P
	synced       bool
	chainStarted bool
}

// NewInitialSync configures the initial sync service responsible for bringing the node up to the
// latest head of the blockchain.
func NewInitialSync(cfg *Config) *InitialSync {
	return &InitialSync{
		chain: cfg.Chain,
		p2p:   cfg.P2P,
	}
}

// Start the initial sync service.
func (s *InitialSync) Start() {
	var genesis time.Time

	// Wait for state to be initialized, if not already.
	ch := make(chan time.Time)
	sub := s.chain.StateInitializedFeed().Subscribe(ch)
	defer sub.Unsubscribe()
	if s.chain.HeadState() == nil {
		// Wait until chain start.
		genesis = <-ch
	} else {
		genesis = time.Unix(int64(s.chain.HeadState().GenesisTime), 0)
	}

	if genesis.After(roughtime.Now()) {
		log.WithField(
			"genesis time",
			genesis,
		).Warn("Genesis time is in the future - waiting to start sync...")
		time.Sleep(roughtime.Until(genesis))
	}
	s.chainStarted = true
	currentSlot := slotsSinceGenesis(genesis)
	if helpers.SlotToEpoch(currentSlot) == 0 {
		log.Info("Chain started within the last epoch - not syncing")
		s.synced = true
		return
	}
	log.Info("Starting initial chain sync...")
	// Are we already in sync, or close to it?
	if helpers.SlotToEpoch(s.chain.HeadSlot()) == helpers.SlotToEpoch(currentSlot) {
		log.Info("Already synced to the current chain head")
		s.synced = true
		return
	}

	// Every 5 sec, report handshake count.
	for {
		count := peerstatus.Count()
		log.WithField(
			"handshakes",
			fmt.Sprintf("%d/%d", count, minStatusCount),
		).Info("Waiting for enough peer handshakes before syncing")

		if count >= minStatusCount {
			break
		}
		time.Sleep(handshakePollingInterval)
	}

	if err := s.roundRobinSync(genesis); err != nil {
		panic(err)
	}

	log.Infof("Synced up to slot %d", s.chain.HeadSlot())
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

// Syncing returns true if initial sync is still running.
func (s *InitialSync) Syncing() bool {
	return !s.synced
}

func slotsSinceGenesis(genesisTime time.Time) uint64 {
	return uint64(roughtime.Since(genesisTime).Seconds()) / params.BeaconConfig().SecondsPerSlot
}
