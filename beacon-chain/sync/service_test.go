package sync

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
)

func NotSyncQuerierConfig() *QuerierConfig {
	return &QuerierConfig{
		ResponseBufferSize: 100,
		CurentHeadSlot:     10,
	}
}

func initializeTestSyncService(ctx context.Context, cfg *Config, synced bool) *Service {
	var sqCfg *QuerierConfig
	if synced {
		sqCfg = DefaultQuerierConfig()
	} else {
		sqCfg = NotSyncQuerierConfig()
	}
	sqCfg.BeaconDB = cfg.BeaconDB
	sqCfg.P2P = cfg.P2P

	isCfg := initialsync.DefaultConfig()
	isCfg.BeaconDB = cfg.BeaconDB
	isCfg.P2P = cfg.P2P

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

func setupTestSyncService(t *testing.T, synced bool) (*Service, *db.BeaconDB) {
	db := internal.SetupDB(t)

	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &Config{
		ChainService:     &mockChainService{},
		P2P:              &mockP2P{},
		BeaconDB:         db,
		OperationService: &mockOperationService{},
	}
	service := initializeTestSyncService(context.Background(), cfg, synced)
	return service, db

}

func TestStatus_ReturnsNoErrorWhenSynced(t *testing.T) {
	serviceSynced, db := setupTestSyncService(t, true)
	defer internal.TeardownDB(t, db)
	if serviceSynced.Status() != nil {
		t.Errorf("Wanted nil, but got %v", serviceSynced.Status())
	}
}

func TestStatus_ReturnsErrorWhenNotSynced(t *testing.T) {
	serviceNotSynced, db := setupTestSyncService(t, false)
	defer internal.TeardownDB(t, db)
	_, querierErr := serviceNotSynced.Querier.IsSynced()
	if serviceNotSynced.Status() != querierErr {
		t.Errorf("Wanted %v, but got %v", querierErr, serviceNotSynced.Status())
	}
}
