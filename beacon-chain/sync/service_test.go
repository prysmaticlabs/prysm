package sync

import (
	"context"
	"testing"
	"time"

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

func setupTestSyncService(t *testing.T, synced bool) *Service {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

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
	return initializeTestSyncService(context.Background(), cfg, synced)
}

func TestStatusWithSyncedNetwork(t *testing.T) {
	ss := setupTestSyncService(t, true)
	_, querierErr := ss.Querier.IsSynced()

	if querierErr != nil {
		t.Errorf("querierErr expected to be nil but got %v", querierErr)
	}
	if statusErr := ss.Status(); statusErr != nil {
		t.Errorf("Expected nil but got %v", statusErr)
	}
}


func TestStatusWithNotSyncedNetwork(t *testing.T) {
	ss := setupTestSyncService(t, false)
	_, querierErr := ss.Querier.IsSynced()

	if querierErr == nil {
		t.Errorf("querierErr expected to be not nil but got %v", querierErr)
		}
	if statusErr := ss.Status(); statusErr != querierErr {
		t.Errorf("Expected %v, but got %v", querierErr, statusErr)
	}
}

