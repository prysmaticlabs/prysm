package sync

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	initialsync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync"
	"github.com/prysmaticlabs/prysm/shared/bls"
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

func setupInitialDeposits(t *testing.T, numDeposits int) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		depositInput := &pb.DepositInput{
			Pubkey: priv.PublicKey().Marshal(),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func setupTestSyncService(t *testing.T, synced bool) (*Service, *db.BeaconDB) {
	db := internal.SetupDB(t)

	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
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
