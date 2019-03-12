package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/sirupsen/logrus"
)

var slog = logrus.WithField("prefix", "sync")

// Service defines the main routines used in the sync service.
type Service struct {
	RegularSync *RegularSync
	InitialSync *initialsync.InitialSync
	Querier     *Querier
}

// Config defines the configured services required for sync to work.
type Config struct {
	ChainService     chainService
	BeaconDB         *db.BeaconDB
	P2P              p2pAPI
	OperationService operationService
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
	isCfg.ChainService = cfg.ChainService

	rsCfg := DefaultRegularSyncConfig()
	rsCfg.ChainService = cfg.ChainService
	rsCfg.BeaconDB = cfg.BeaconDB
	rsCfg.P2P = cfg.P2P

	sq := NewQuerierService(ctx, sqCfg)
	rs := NewRegularSyncService(ctx, rsCfg)

	isCfg.SyncService = rs
	is := initialsync.NewInitialSyncService(ctx, isCfg)

	return &Service{
		RegularSync: rs,
		InitialSync: is,
		Querier:     sq,
	}

}

// Start kicks off the sync service
func (ss *Service) Start() {
	slog.Info("Starting Sync Service")
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
	synced, err := ss.Querier.IsSynced()
	if !synced && err != nil {
		return err
	}
	return nil
}

func (ss *Service) run() {
	ss.Querier.Start()
	synced, err := ss.Querier.IsSynced()
	if err != nil {
		slog.Fatalf("Unable to retrieve result from sync querier %v", err)
	}

	// return since there is nothing to sync
	if !ss.Querier.chainStarted && ss.Querier.atGenesis {
		return
	}

	if synced {
		ss.RegularSync.Start()
		return
	}

	// Sets the highest observed slot from querier.
	ss.InitialSync.InitializeObservedSlot(ss.Querier.currentHeadSlot)

	// Sets the state root of the highest observed slot.
	ss.InitialSync.InitializeStateRoot(ss.Querier.currentStateRoot)

	ss.InitialSync.Start()
}
