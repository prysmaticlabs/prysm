package sync

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

var _ = Checker(&Service{})

func NotSyncQuerierConfig() *QuerierConfig {
	return &QuerierConfig{
		ResponseBufferSize: 100,
		CurrentHeadSlot:    10,
	}
}

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{})
}

func initializeTestSyncService(ctx context.Context, cfg *Config, synced bool) *Service {
	var sqCfg *QuerierConfig
	if synced {
		sqCfg = DefaultQuerierConfig()
	} else {
		sqCfg = NotSyncQuerierConfig()
	}

	services := NewSyncService(ctx, cfg)

	sqCfg.BeaconDB = cfg.BeaconDB
	sqCfg.P2P = cfg.P2P
	sq := NewQuerierService(ctx, sqCfg)

	services.Querier = sq

	return services
}

func setupTestSyncService(t *testing.T, synced bool) (*Service, *db.BeaconDB) {
	db := internal.SetupDBDeprecated(t)

	unixTime := uint64(time.Now().Unix())
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), unixTime, deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &Config{
		ChainService: &mockChainService{
			db: db,
		},
		P2P:              &mockP2P{},
		BeaconDB:         db,
		OperationService: &mockOperationService{},
	}
	service := initializeTestSyncService(context.Background(), cfg, synced)
	return service, db

}

func TestStatus_NotSynced(t *testing.T) {
	serviceNotSynced, db := setupTestSyncService(t, false)
	defer internal.TeardownDBDeprecated(t, db)
	synced := serviceNotSynced.InitialSync.NodeIsSynced()
	if synced {
		t.Error("Wanted false, but got true")
	}
}
