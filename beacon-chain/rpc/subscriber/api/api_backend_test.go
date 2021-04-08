package api

import (
	"context"
	"encoding/binary"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/beacon"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api/events"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestAPIBackend_SubscribeNewEpochEvent(t *testing.T) {
	consensusChannel := make(chan interface{})
	helpers.ClearCache()
	db := dbTest.SetupDB(t)
	ctx := context.Background()

	config := params.BeaconConfig().Copy()
	oldConfig := config.Copy()
	config.SlotsPerEpoch = 32
	params.OverrideBeaconConfig(config)

	defer func() {
		params.OverrideBeaconConfig(oldConfig)
	}()
	testStartTime := time.Now()

	stateNotifier := new(mock.ChainService).StateNotifier()

	count := 10000
	validators := make([]*ethpb.Validator, 0, count)
	withdrawCred := make([]byte, 32)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		val := &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: withdrawCred,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		validators = append(validators, val)
	}

	blk := testutil.NewBeaconBlock().Block
	parentRoot := [32]byte{1, 2, 3}
	blk.ParentRoot = parentRoot[:]
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	state, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, state.SetSlot(0))

	bs := &beacon.Server{
		BeaconDB: db,
		Ctx:      context.Background(),
		FinalizationFetcher: &mock.ChainService{
			Genesis: testStartTime,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			Genesis: testStartTime,
		},
		StateGen:      stategen.New(db),
		StateNotifier: stateNotifier,
		HeadFetcher: &mock.ChainService{
			State: state,
		},
	}

	require.NoError(t, state.SetValidators(validators))
	require.NoError(t, db.SaveState(bs.Ctx, state, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(bs.Ctx, blockRoot))

	stateFeed := stateNotifier.StateFeed()

	apiBackend := APIBackend{
		BeaconChain: *bs,
	}

	apiBackend.SubscribeNewEpochEvent(ctx, types.Epoch(0), consensusChannel)
	shouldGather := 1
	received := make([]*events.MinimalEpochConsensusInfo, 0)

	var sendWaitGroup sync.WaitGroup
	sendWaitGroup.Add(shouldGather)

	go func() {
		for {
			consensusInfo := <-consensusChannel

			if nil != consensusInfo {
				received = append(received, consensusInfo.(*events.MinimalEpochConsensusInfo))
				sendWaitGroup.Done()
			}
		}
	}()

	ticker := time.NewTicker(time.Second * 5)
	sent := 0
	sendUntilTimeout := func(shouldSend bool) {
		for sent == 0 {
			select {
			case <-ticker.C:
				t.FailNow()
			default:
				if !shouldSend {
					return
				}
				sent = stateFeed.Send(&feed.Event{
					Type: statefeed.BlockProcessed,
					Data: &statefeed.BlockProcessedData{},
				})
			}
		}
	}

	t.Run("Should not send because epoch not increased", func(t *testing.T) {
		sendUntilTimeout(true)
		assert.Equal(t, 1, sent)
		assert.Equal(t, 0, len(received))
	})

	t.Run("Should send because epoch increased", func(t *testing.T) {
		require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch))
		sendUntilTimeout(true)
		sendWaitGroup.Wait()
		assert.Equal(t, 1, sent)
		assert.Equal(t, shouldGather, len(received))
	})
}
