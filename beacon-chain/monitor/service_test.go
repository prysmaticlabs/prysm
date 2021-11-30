package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func setupService(t *testing.T) *Service {
	beaconDB := testDB.SetupDB(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 256)

	pubKeys := make([][]byte, 3)
	pubKeys[0] = state.Validators()[0].PublicKey
	pubKeys[1] = state.Validators()[1].PublicKey
	pubKeys[2] = state.Validators()[2].PublicKey

	currentSyncCommittee := util.ConvertToCommittee([][]byte{
		pubKeys[0], pubKeys[1], pubKeys[2], pubKeys[1], pubKeys[1],
	})
	require.NoError(t, state.SetCurrentSyncCommittee(currentSyncCommittee))

	chainService := &mock.ChainService{
		Genesis:        time.Now(),
		DB:             beaconDB,
		State:          state,
		Root:           []byte("hello-world"),
		ValidatorsRoot: [32]byte{},
	}

	trackedVals := map[types.ValidatorIndex]interface{}{
		1:  nil,
		2:  nil,
		12: nil,
		15: nil,
	}
	latestPerformance := map[types.ValidatorIndex]ValidatorLatestPerformance{
		1: {
			balance: 32000000000,
		},
		2: {
			balance: 32000000000,
		},
		12: {
			balance: 31900000000,
		},
		15: {
			balance: 31900000000,
		},
	}
	aggregatedPerformance := map[types.ValidatorIndex]ValidatorAggregatedPerformance{
		1:  {},
		2:  {},
		12: {},
		15: {},
	}
	trackedSyncCommitteeIndices := map[types.ValidatorIndex][]types.CommitteeIndex{
		1:  {0, 1, 2, 3},
		12: {4, 5},
	}
	return &Service{
		config: &ValidatorMonitorConfig{
			StateGen:            stategen.New(beaconDB),
			StateNotifier:       chainService.StateNotifier(),
			HeadFetcher:         chainService,
			AttestationNotifier: chainService.OperationNotifier(),
		},

		ctx:                         context.Background(),
		TrackedValidators:           trackedVals,
		latestPerformance:           latestPerformance,
		aggregatedPerformance:       aggregatedPerformance,
		trackedSyncCommitteeIndices: trackedSyncCommitteeIndices,
		lastSyncedEpoch:             0,
	}
}

func TestTrackedIndex(t *testing.T) {
	s := &Service{
		TrackedValidators: map[types.ValidatorIndex]interface{}{
			1: nil,
			2: nil,
		},
	}
	require.Equal(t, s.trackedIndex(types.ValidatorIndex(1)), true)
	require.Equal(t, s.trackedIndex(types.ValidatorIndex(3)), false)
}

func TestUpdateSyncCommitteeTrackedVals(t *testing.T) {
	hook := logTest.NewGlobal()
	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 1024)

	s.updateSyncCommitteeTrackedVals(state)
	require.LogsDoNotContain(t, hook, "sync committee assignments will not be reported")
	newTrackedSyncIndices := map[types.ValidatorIndex][]types.CommitteeIndex{
		1: {1, 3, 4},
		2: {2},
	}
	require.DeepEqual(t, s.trackedSyncCommitteeIndices, newTrackedSyncIndices)
}

func TestNewService(t *testing.T) {
	config := &ValidatorMonitorConfig{}
	tracked := []types.ValidatorIndex{}
	ctx := context.Background()
	_, err := NewService(ctx, config, tracked)
	require.NoError(t, err)
}

func TestStart(t *testing.T) {
	hook := logTest.NewGlobal()
	s := setupService(t)
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.config.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	s.Start()

	go func() {
		select {
		case stateEvent := <-stateChannel:
			if stateEvent.Type == statefeed.Synced {
				_, ok := stateEvent.Data.(*statefeed.SyncedData)
				require.Equal(t, true, ok, "event feed data is not type *statefeed.SyncedData")
			}
		case <-s.ctx.Done():
		}
		wg.Done()
	}()

	for sent := 0; sent == 0; {
		sent = s.config.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Synced,
			Data: &statefeed.SyncedData{
				StartTime: time.Now(),
			},
		})
	}

	time.Sleep(2000 * time.Millisecond) // wait for updateSyncCommitteeTrackedVals
	require.LogsContain(t, hook, "\"Started service\" ValidatorIndices=\"[1 2 12 15]\"")
	require.Equal(t, s.isRunning, true, "monitor is not running")
}
