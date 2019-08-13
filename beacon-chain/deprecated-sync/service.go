package sync

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/deprecated-sync/initial-sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/sirupsen/logrus"
)

var slog = logrus.WithField("prefix", "sync")

// Checker defines a struct which can verify whether a node is currently
// synchronizing a chain with the rest of peers in the network.
type Checker interface {
	Syncing() bool
}

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
	synced, err := ss.Querier.IsSynced()
	if err != nil {
		return err
	}

	if synced {
		return nil
	}

	if !ss.InitialSync.NodeIsSynced() {
		return errors.New("not initially synced")
	}
	return nil
}

// Syncing verifies the sync service is fully synced with the network
// and returns the result as a boolean.
func (ss *Service) Syncing() bool {
	querierSynced, err := ss.Querier.IsSynced()
	if err != nil {
		return false
	}
	isSynced := querierSynced && ss.InitialSync.NodeIsSynced()
	return !isSynced
}

func (ss *Service) run() {
	ss.Querier.Start()

	synced, err := ss.Querier.IsSynced()
	if err != nil {
		slog.Fatalf("Unable to retrieve result from sync querier %v", err)
	}
	ss.querierFinished = true

	if synced {
		ss.RegularSync.Start()
		return
	}

	ss.InitialSync.Start(ss.Querier.chainHeadResponses)
}
