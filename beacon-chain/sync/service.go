package sync

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&RegularSync{})

// Config to set up the regular sync service.
type Config struct {
	P2P         p2p.P2P
	DB          db.Database
	Operations  *operations.Service
	Chain       blockchainService
	InitialSync Checker
}

// This defines the interface for interacting with block chain service
type blockchainService interface {
	blockchain.BlockReceiver
	blockchain.HeadFetcher
	blockchain.FinalizationFetcher
	blockchain.AttestationReceiver
	blockchain.ChainFeeds
	blockchain.GenesisTimeFetcher
}

// NewRegularSync service.
func NewRegularSync(cfg *Config) *RegularSync {
	r := &RegularSync{
		ctx:                 context.Background(),
		db:                  cfg.DB,
		p2p:                 cfg.P2P,
		operations:          cfg.Operations,
		chain:               cfg.Chain,
		initialSync:         cfg.InitialSync,
		slotToPendingBlocks: make(map[uint64]*ethpb.BeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
	}

	r.registerRPCHandlers()
	r.registerSubscribers()

	return r
}

// RegularSync service is responsible for handling all run time p2p related operations as the
// main entry point for network messages.
type RegularSync struct {
	ctx                 context.Context
	p2p                 p2p.P2P
	db                  db.Database
	operations          *operations.Service
	chain               blockchainService
	slotToPendingBlocks map[uint64]*ethpb.BeaconBlock
	seenPendingBlocks   map[[32]byte]bool
	pendingQueueLock    sync.RWMutex
	chainStarted        bool
	initialSync         Checker
	validateBlockLock   sync.RWMutex
}

// Start the regular sync service.
func (r *RegularSync) Start() {
	r.p2p.AddConnectionHandler(r.sendRPCStatusRequest)
	r.p2p.AddDisconnectionHandler(r.removeDisconnectedPeerStatus)
	go r.processPendingBlocksQueue()
	go r.maintainPeerStatuses()
}

// Stop the regular sync service.
func (r *RegularSync) Stop() error {
	return nil
}

// Status of the currently running regular sync service.
func (r *RegularSync) Status() error {
	if r.chainStarted && r.initialSync.Syncing() {
		return errors.New("waiting for initial sync")
	}
	return nil
}

// Checker defines a struct which can verify whether a node is currently
// synchronizing a chain with the rest of peers in the network.
type Checker interface {
	Syncing() bool
	Status() error
}

// StatusTracker interface for accessing the status / handshake messages received so far.
type StatusTracker interface {
	PeerStatuses() map[peer.ID]*pb.Status
}
