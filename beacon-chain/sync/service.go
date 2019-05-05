package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var slog = logrus.WithField("prefix", "sync")

// Service defines the main routines used in the sync service.
type Service struct {
	RegularSync     *RegularSync
	InitialSync     *initialsync.InitialSync
	Querier         *Querier
	querierFinished bool
}

// Config defines the configured services required for sync to work.
type Config struct {
	ChainService     chainService
	BeaconDB         *db.BeaconDB
	P2P              p2pAPI
	AttsService      attsService
	OperationService operations.OperationFeeds
	PowChainService  powChainService
}

// NewSyncService creates a new instance of SyncService using the config
// given.
func NewSyncService(ctx context.Context, cfg *Config) *Service {
	sqCfg := DefaultQuerierConfig()
	sqCfg.BeaconDB = cfg.BeaconDB
	sqCfg.P2P = cfg.P2P
	sqCfg.PowChain = cfg.PowChainService
	sqCfg.ChainService = cfg.ChainService

	isCfg := initialsync.DefaultConfig()
	isCfg.BeaconDB = cfg.BeaconDB
	isCfg.P2P = cfg.P2P
	isCfg.PowChain = cfg.PowChainService
	isCfg.ChainService = cfg.ChainService

	rsCfg := DefaultRegularSyncConfig()
	rsCfg.ChainService = cfg.ChainService
	rsCfg.BeaconDB = cfg.BeaconDB
	rsCfg.P2P = cfg.P2P
	rsCfg.AttsService = cfg.AttsService
	rsCfg.OperationService = cfg.OperationService

	sq := NewQuerierService(ctx, sqCfg)
	rs := NewRegularSyncService(ctx, rsCfg)

	isCfg.SyncService = rs
	is := initialsync.NewInitialSyncService(ctx, isCfg)

	return &Service{
		RegularSync:     rs,
		InitialSync:     is,
		Querier:         sq,
		querierFinished: false,
	}

}

// Start kicks off the sync service
func (ss *Service) Start() {
	slog.Info("Starting service")
	go ss.run()
}

// Stop ends all the currently running routines
// which are part of the sync service.
func (ss *Service) Stop() error {
	err := ss.Querier.Stop()
	if err != nil {
		return err
	}

	err = ss.InitialSync.Stop()
	if err != nil {
		return err
	}
	return ss.RegularSync.Stop()
}

// Status checks the status of the node. It returns nil if it's synced
// with the rest of the network and no errors occurred. Otherwise, it returns an error.
func (ss *Service) Status() error {
	if !ss.querierFinished && !ss.Querier.atGenesis {
		return errors.New("querier is still running")
	}

	if !ss.Querier.chainStarted {
		return nil
	}

	if ss.Querier.atGenesis {
		return nil
	}

	blk, err := ss.Querier.db.ChainHead()
	if err != nil {
		return fmt.Errorf("could not retrieve chain head %v", err)
	}
	if blk == nil {
		return errors.New("no chain head exists in db")
	}
	if blk.Slot < ss.InitialSync.HighestObservedSlot() {
		return fmt.Errorf("node is not synced as the current chain head is at slot %d", blk.Slot-params.BeaconConfig().GenesisSlot)
	}
	return nil
}

func (ss *Service) run() {
	ss.Querier.Start()
	synced, err := ss.Querier.IsSynced()
	if err != nil {
		slog.Fatalf("Unable to retrieve result from sync querier %v", err)
	}
	ss.querierFinished = true

	// Sets the highest observed slot from querier.
	ss.InitialSync.InitializeObservedSlot(ss.Querier.currentHeadSlot)
	ss.InitialSync.InitializeBestPeer(ss.Querier.bestPeer)
	ss.InitialSync.InitializeObservedStateRoot(bytesutil.ToBytes32(ss.Querier.currentStateRoot))
	// Sets the state root of the highest observed slot.
	ss.InitialSync.InitializeFinalizedStateRoot(ss.Querier.currentFinalizedStateRoot)

	if synced {
		ss.RegularSync.Start()
		return
	}

	ss.InitialSync.Start()
}
