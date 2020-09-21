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
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
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
	hook := logTest.NewGlobal()
	tests := []struct {
		name         string
		assert       func()
		chainService func() *mock.ChainService
	}{
		{
			name: "head is not ready",
			assert: func() {
				assert.LogsContain(t, hook, "Waiting for state to be initialized")
			},
		},
		{
			name: "future genesis",
			chainService: func() *mock.ChainService {
				// Set to future time (genesis time hasn't arrived yet).
				st := testutil.NewBeaconState()
				require.NoError(t, st.SetGenesisTime(uint64(time.Unix(4113849600, 0).Unix())))
				return &mock.ChainService{
					State: st,
					FinalizedCheckPoint: &eth.Checkpoint{
						Epoch: 0,
					},
				}
			},
			assert: func() {
				assert.LogsContain(t, hook, "Genesis time has not arrived - not syncing")
				assert.LogsDoNotContain(t, hook, "Waiting for state to be initialized")
			},
		},
		{
			name: "current epoch",
			chainService: func() *mock.ChainService {
				// Set to nearby slot.
				st := testutil.NewBeaconState()
				require.NoError(t, st.SetGenesisTime(uint64(time.Now().Add(-5*time.Minute).Unix())))
				return &mock.ChainService{
					State: st,
					FinalizedCheckPoint: &eth.Checkpoint{
						Epoch: 0,
					},
				}
			},
			assert: func() {
				assert.LogsContain(t, hook, "Chain started within the last epoch - not syncing")
				assert.LogsDoNotContain(t, hook, "Genesis time has not arrived - not syncing")
				assert.LogsDoNotContain(t, hook, "Waiting for state to be initialized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer hook.Reset()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mc := &mock.ChainService{}
			// Allow overriding with customized chain service.
			if tt.chainService != nil {
				mc = tt.chainService()
			}
			s := NewInitialSync(ctx, &Config{
				Chain:         mc,
				StateNotifier: mc.StateNotifier(),
			})
			assert.NotNil(t, s)

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s.Start()
				wg.Done()
			}()
			go func() {
				// Allow to exit from test (on no head loop waiting for head is started).
				// In most tests, this is redundant, as Start() already exited.
				time.AfterFunc(500*time.Millisecond, func() {
					cancel()
				})
			}()
			if testutil.WaitTimeout(wg, time.Second*2) {
				t.Fatalf("Test should have exited by now, timed out")
			}
			tt.assert()
		})
	}
}

func TestService_waitForStateInitialization(t *testing.T) {
	hook := logTest.NewGlobal()
	newService := func(ctx context.Context, mc *mock.ChainService) *Service {
		s := NewInitialSync(ctx, &Config{
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
			FinalizedCheckPoint: &eth.Checkpoint{
				Epoch: 0,
			},
		}
		s := newService(ctx, mc)

		expectedGenesisTime := time.Unix(25000, 0)
		var receivedGenesisTime time.Time
		require.NoError(t, mc.State.SetGenesisTime(uint64(expectedGenesisTime.Unix())))
		receivedGenesisTime, err := s.waitForStateInitialization()
		assert.NoError(t, err)
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
			_, err := s.waitForStateInitialization()
			assert.ErrorContains(t, "context closed", err)
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

		expectedGenesisTime := time.Unix(358544700, 0)
		var receivedGenesisTime time.Time
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			var err error
			receivedGenesisTime, err = s.waitForStateInitialization()
			assert.NoError(t, err)
			wg.Done()
		}()
		go func() {
			time.AfterFunc(500*time.Millisecond, func() {
				// Send invalid event at first.
				s.stateNotifier.StateFeed().Send(&feed.Event{
					Type: statefeed.Initialized,
					Data: &statefeed.BlockProcessedData{},
				})
				// Send valid event.
				s.stateNotifier.StateFeed().Send(&feed.Event{
					Type: statefeed.Initialized,
					Data: &statefeed.InitializedData{
						StartTime:             expectedGenesisTime,
						GenesisValidatorsRoot: make([]byte, 32),
					},
				})
			})
		}()

		if testutil.WaitTimeout(wg, time.Second*2) {
			t.Fatalf("Test should have exited by now, timed out")
		}
		assert.Equal(t, expectedGenesisTime, receivedGenesisTime)
		assert.LogsContain(t, hook, "Event feed data is not type *statefeed.InitializedData")
		assert.LogsContain(t, hook, "Waiting for state to be initialized")
		assert.LogsContain(t, hook, "Received state initialized event")
		assert.LogsDoNotContain(t, hook, "Context closed, exiting goroutine")
	})
}
