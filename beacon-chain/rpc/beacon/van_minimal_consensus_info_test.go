package beacon

import (
	"context"
	"encoding/binary"
	"github.com/golang/mock/gomock"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	vmock "github.com/prysmaticlabs/prysm/shared/van_mock"
	"testing"
	"time"
)

func TestServer_StreamMinimalConsensusInfo(t *testing.T) {
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

	stateNotifier := new(chainMock.ChainService).StateNotifier()

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
	beaconState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(0))

	server := &Server{
		BeaconDB: db,
		Ctx:      ctx,
		FinalizationFetcher: &chainMock.ChainService{
			Genesis: testStartTime,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &chainMock.ChainService{
			Genesis: testStartTime,
		},
		StateGen:      stategen.New(db),
		StateNotifier: stateNotifier,
		HeadFetcher: &chainMock.ChainService{
			State: beaconState,
		},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	require.NoError(t, beaconState.SetValidators(validators))
	require.NoError(t, db.SaveState(server.Ctx, beaconState, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(server.Ctx, blockRoot))

	wantedRes, err := server.MinimalConsensusInfoRange(server.Ctx, types.Epoch(0))
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exitRoutine := make(chan bool)
	mockStream := vmock.NewMockBeaconChain_StreamMinimalConsensusInfoServer(ctrl)
	mockStream.EXPECT().Send(wantedRes).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()
	minimalConsensusInfoRequest := &ethpb.MinimalConsensusInfoRequest{FromEpoch: types.Epoch(0)}

	go func(tt *testing.T) {
		assert.NoError(t, server.StreamMinimalConsensusInfo(minimalConsensusInfoRequest, mockStream), "Could not call RPC method")
	}(t)

	<-exitRoutine
}
