package initialsync

import (
	"context"
	"sync"
	"testing"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_Constants(t *testing.T) {
	if params.BeaconConfig().MaxPeersToSync*flags.Get().BlockBatchLimit > 1000 {
		t.Fatal("rpc rejects requests over 1000 range slots")
	}
}

func TestService_InitStartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mc, p2p, db := initializeTestServices(t, []uint64{}, []*peerData{})
	s := NewInitialSync(ctx, &Config{
		P2P:   p2p,
		DB:    db,
		Chain: mc,
	})
	assert.NotNil(t, s)
}

func TestService_waitForStateInitialization(t *testing.T) {
	hook := logTest.NewGlobal()

	// Setup database.
	beaconDB, _ := dbtest.SetupDB(t)
	genesisBlk := testutil.NewBeaconBlock()
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	err = beaconDB.SaveBlock(context.Background(), genesisBlk)
	require.NoError(t, err)

	newService := func(ctx context.Context, mc *mock.ChainService) *Service {
		s := NewInitialSync(ctx, &Config{
			P2P:           p2pt.NewTestP2P(t),
			DB:            beaconDB,
			Chain:         mc,
			StateNotifier: mc.StateNotifier(),
		})
		require.NotNil(t, s)
		return s
	}

	t.Run("head state exists", func(t *testing.T) {
		defer hook.Reset()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := &mock.ChainService{
			State: testutil.NewBeaconState(),
			Root:  genesisBlkRoot[:],
			DB:    beaconDB,
			FinalizedCheckPoint: &eth.Checkpoint{
				Epoch: 0,
			},
		}
		s := newService(ctx, mc)

		expectedGenesisTime := time.Unix(25000, 0)
		var receivedGenesisTime time.Time
		require.NoError(t, mc.State.SetGenesisTime(uint64(expectedGenesisTime.Unix())))

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			receivedGenesisTime = s.waitForStateInitialization()
			wg.Done()
		}()

		if testutil.WaitTimeout(wg, time.Second*2) {
			t.Fatalf("Test should have exited by now, timed out")
		}
		assert.Equal(t, expectedGenesisTime, receivedGenesisTime)
		assert.LogsDoNotContain(t, hook, "Waiting for state to be initialized")
	})

	t.Run("no state and context close", func(t *testing.T) {
		defer hook.Reset()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s := newService(ctx, &mock.ChainService{})
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			s.waitForStateInitialization()
			wg.Done()
		}()
		go func() {
			time.AfterFunc(500*time.Millisecond, func() {
				cancel()
			})
		}()

		if testutil.WaitTimeout(wg, time.Second*2) {
			t.Fatalf("Test should have exited by now, timed out")
		}
		assert.LogsContain(t, hook, "Waiting for state to be initialized")
		assert.LogsContain(t, hook, "Context closed, exiting goroutine")
		assert.LogsDoNotContain(t, hook, "Subscription to state notifier failed")
	})

	t.Run("no state and state init event received", func(t *testing.T) {
		defer hook.Reset()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s := newService(ctx, &mock.ChainService{})

		genesisTime := time.Unix(358544700, 0)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			assert.Equal(t, genesisTime, s.waitForStateInitialization())
			wg.Done()
		}()
		go func() {
			time.AfterFunc(500*time.Millisecond, func() {
				s.stateNotifier.StateFeed().Send(&feed.Event{
					Type: statefeed.Initialized,
					Data: &statefeed.InitializedData{
						StartTime:             genesisTime,
						GenesisValidatorsRoot: make([]byte, 32),
					},
				})
			})
		}()

		if testutil.WaitTimeout(wg, time.Second*2) {
			t.Fatalf("Test should have exited by now, timed out")
		}
		assert.LogsContain(t, hook, "Waiting for state to be initialized")
		assert.LogsContain(t, hook, "Received state initialized event")
		assert.LogsDoNotContain(t, hook, "Context closed, exiting goroutine")
	})
}
