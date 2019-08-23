package sync

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&RegularSync{})

// Config to set up the regular sync service.
type Config struct {
	P2P        p2p.P2P
	DB         db.Database
	Operations *operations.Service
	Chain      blockchainService
}

// This defines the interface for interacting with block chain service
type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.HeadRetriever
	blockchain.FinalizationRetriever
}

// NewRegularSync service.
func NewRegularSync(cfg *Config) *RegularSync {
	return &RegularSync{
		ctx:        context.Background(),
		db:         cfg.DB,
		p2p:        cfg.P2P,
		operations: cfg.Operations,
		chain:      cfg.Chain,
	}
}

// RegularSync service is responsible for handling all run time p2p related operations as the
// main entry point for network messages.
type RegularSync struct {
	ctx        context.Context
	p2p        p2p.P2P
	db         db.Database
	operations *operations.Service
	chain      blockchainService
}

// Start the regular sync service by initializing all of the p2p sync handlers.
func (r *RegularSync) Start() {
	log.Info("Starting regular sync")
	for !r.p2p.Started() {
		time.Sleep(200 * time.Millisecond)
	}
	r.registerRPCHandlers()
	r.registerSubscribers()
	log.Info("Regular sync started")
}

// Stop the regular sync service.
func (r *RegularSync) Stop() error {
	return nil
}

// Status of the currently running regular sync service.
func (r *RegularSync) Status() error {
	return nil
}

// Syncing returns true if the node is currently syncing with the network.
func (r *RegularSync) Syncing() bool {
	// TODO(3147): Use real value.
	return false
}

// Checker defines a struct which can verify whether a node is currently
// synchronizing a chain with the rest of peers in the network.
type Checker interface {
	Syncing() bool
	Status() error
}
