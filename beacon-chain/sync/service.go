package sync

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/sirupsen/logrus"
)

var slog = logrus.WithField("prefix", "sync")

// SyncService defines the main routines used in the sync service.
type SyncService struct {
	RegularSync *RegularSync
	InitialSync *initialsync.InitialSync
	Querier     *SyncQuerier
}

// SyncConfig defines the configured services required for sync to work.
type SyncConfig struct {
	ChainService  chainService
	AttestService attestationService
	BeaconDB      *db.BeaconDB
	P2P           p2pAPI
}

// NewSyncService creates a new instance of SyncService using the config
// given.
func NewSyncService(ctx context.Context, cfg *SyncConfig) *SyncService {

	sqCfg := DefaultQuerierConfig()
	sqCfg.BeaconDB = cfg.BeaconDB
	sqCfg.P2P = cfg.P2P

	isCfg := initialsync.DefaultConfig()
	isCfg.BeaconDB = cfg.BeaconDB
	isCfg.P2P = cfg.P2P

	rsCfg := DefaultRegularSyncConfig()
	rsCfg.ChainService = cfg.ChainService
	rsCfg.AttestService = cfg.AttestService
	rsCfg.BeaconDB = cfg.BeaconDB
	rsCfg.P2P = cfg.P2P

	sq := NewSyncQuerierService(ctx, sqCfg)
	rs := NewRegularSyncService(ctx, rsCfg)

	isCfg.SyncService = rs
	is := initialsync.NewInitialSyncService(ctx, isCfg)

	return &SyncService{
		RegularSync: rs,
		InitialSync: is,
		Querier:     sq,
	}

}

// Start kicks off the sync service
func (ss *SyncService) Start() {
	go ss.run()
}

// Stop ends all the currently running routines
// which are part of the sync service
func (ss *SyncService) Stop() error {
	synced, err := ss.Querier.IsSynced()
	if err != nil {
		return err
	}
	if synced {
		err := ss.RegularSync.Stop()
		return err
	}

	return ss.InitialSync.Stop()
}

func (ss *SyncService) run() {
	ss.Querier.Start()
	synced, err := ss.Querier.IsSynced()
	if err != nil {
		slog.Fatalf("Unable to retrieve result from sync querier %v", err)
	}

	if synced {
		ss.RegularSync.Start()
		return
	}

	ss.InitialSync.Start()
}
