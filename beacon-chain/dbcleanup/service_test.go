package dbcleanup

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockChainService struct {
	stateFeed *event.Feed
}

func (m *mockChainService) CanonicalStateFeed() *event.Feed {
	return m.stateFeed
}

func newMockChainService() *mockChainService {
	return &mockChainService{
		stateFeed: new(event.Feed),
	}
}

func setupInitialDeposits(t *testing.T) []*pb.Deposit {
	genesisValidatorRegistry := validators.InitialValidatorRegistry()
	deposits := make([]*pb.Deposit, len(genesisValidatorRegistry))
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: genesisValidatorRegistry[i].Pubkey,
		}
		balance := params.BeaconConfig().MaxDeposit
		depositData, err := blocks.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	return deposits
}

func createCleanupService(beaconDB *db.BeaconDB) *CleanupService {
	chainService := newMockChainService()

	cleanupService := NewCleanupService(context.Background(), &Config{
		SubscriptionBuf: 100,
		BeaconDB:        beaconDB,
		ChainService:    chainService,
	})
	return cleanupService
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()

	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	cleanupService := createCleanupService(beaconDB)

	cleanupService.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")

	if err := cleanupService.Stop(); err != nil {
		t.Fatalf("failed to stop cleanup service: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestCleanBlockVoteCache(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	var err error

	// Pre-fill block vote cache in DB
	genesisTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err = beaconDB.InitializeState(genesisTime, deposits); err != nil {
		t.Fatalf("failed to initialize DB: %v", err)
	}
	oldBlock := &pb.BeaconBlock{Slot: 1}
	oldBlockHash, _ := hashutil.HashBeaconBlock(oldBlock)
	if err = beaconDB.SaveBlock(oldBlock); err != nil {
		t.Fatalf("failed to write block int DB: %v", err)
	}
	oldState := &pb.BeaconState{}
	if err = beaconDB.SaveState(oldState); err != nil {
		t.Fatalf("failed to pre-fill DB: %v", err)
	}
	if err := beaconDB.UpdateChainHead(oldBlock, oldState); err != nil {
		t.Fatalf("failed to update chain head: %v", err)
	}
	oldBlockVoteCache := utils.NewBlockVoteCache()
	oldBlockVoteCache[oldBlockHash] = utils.NewBlockVote()
	if err = beaconDB.WriteBlockVoteCache(oldBlockVoteCache); err != nil {
		t.Fatalf("failed to write block vote cache into DB: %v", err)
	}

	// Verify block vote cache is not cleaned before running the cleanup service
	blockHashes := [][32]byte{oldBlockHash}
	var blockVoteCache utils.BlockVoteCache
	if blockVoteCache, err = beaconDB.ReadBlockVoteCache(blockHashes); err != nil {
		t.Fatalf("failed to read block vote cache from DB: %v", err)
	}
	if len(blockVoteCache) != 1 {
		t.Fatalf("failed to reach pre-filled block vote cache status")
	}

	// Now let the cleanup service do its job
	cleanupService := createCleanupService(beaconDB)
	state := &pb.BeaconState{FinalizedEpoch: 1}
	if err = cleanupService.cleanBlockVoteCache(state.FinalizedEpoch); err != nil {
		t.Fatalf("failed to clean block vote cache")
	}

	// Check the block vote cache has been cleaned up
	if blockVoteCache, err = beaconDB.ReadBlockVoteCache(blockHashes); err != nil {
		t.Errorf("failed to read block vote cache from DB: %v", err)
	}
	if len(blockVoteCache) != 0 {
		t.Error("block vote cache is expected to be cleaned up")
	}
}
