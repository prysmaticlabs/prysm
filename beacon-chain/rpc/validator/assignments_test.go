package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// pubKey is a helper to generate a well-formed public key.
func pubKey(i uint64) []byte {
	pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey, i)
	return pubKey
}
func TestGetDuties_NextEpoch_CantFindValidatorIdx(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	beaconState, _ := testutil.DeterministicGenesisState(t, 10)

	genesis := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")

	height := time.Unix(int64(params.BeaconConfig().Eth1FollowDistance), 0).Unix()
	p := &mockPOW.POWChain{
		TimesByHeight: map[int]uint64{
			0: uint64(height),
		},
	}

	chain := &mockChain.ChainService{
		State: beaconState, Root: genesisRoot[:], Genesis: time.Now(),
	}
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	vs := &Server{
		BeaconDB:           db,
		HeadFetcher:        chain,
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		Eth1InfoFetcher:    p,
		DepositFetcher:     depositCache,
		GenesisTimeFetcher: chain,
	}

	pubKey := pubKey(99999)
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{pubKey},
	}
	want := fmt.Sprintf("validator %#x does not exist", req.PublicKeys[0])
	if _, err := vs.GetDuties(ctx, req); err != nil && !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestGetDuties_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	genesis := testutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	require.NoError(t, err, "Could not setup genesis bs")
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")

	pubKeys := make([][]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
		indices[i] = uint64(i)
	}

	pubkeysAs48ByteType := make([][48]byte, len(pubKeys))
	for i, pk := range pubKeys {
		pubkeysAs48ByteType[i] = bytesutil.ToBytes48(pk)
	}

	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Genesis: time.Now(),
	}
	vs := &Server{
		BeaconDB:           db,
		HeadFetcher:        chain,
		GenesisTimeFetcher: chain,
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
	}

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
	}
	res, err := vs.GetDuties(context.Background(), req)
	require.NoError(t, err, "Could not call epoch committee assignment")
	if res.CurrentEpochDuties[0].AttesterSlot > bs.Slot()+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.CurrentEpochDuties[0].AttesterSlot, bs.Slot()+params.BeaconConfig().SlotsPerEpoch)
	}

	// Test the last validator in registry.
	lastValidatorIndex := depChainStart - 1
	req = &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[lastValidatorIndex].Data.PublicKey},
	}
	res, err = vs.GetDuties(context.Background(), req)
	require.NoError(t, err, "Could not call epoch committee assignment")
	if res.CurrentEpochDuties[0].AttesterSlot > bs.Slot()+params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Assigned slot %d can't be higher than %d",
			res.CurrentEpochDuties[0].AttesterSlot, bs.Slot()+params.BeaconConfig().SlotsPerEpoch)
	}

	// We request for duties for all validators.
	req = &ethpb.DutiesRequest{
		PublicKeys: pubKeys,
		Epoch:      0,
	}
	res, err = vs.GetDuties(context.Background(), req)
	require.NoError(t, err, "Could not call epoch committee assignment")
	for i := 0; i < len(res.CurrentEpochDuties); i++ {
		assert.Equal(t, uint64(i), res.CurrentEpochDuties[i].ValidatorIndex)
	}
}

func TestGetDuties_CurrentEpoch_ShouldNotFail(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	genesis := testutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	require.NoError(t, err, "Could not setup genesis state")
	// Set state to non-epoch start slot.
	require.NoError(t, bState.SetSlot(5))

	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")

	pubKeys := make([][48]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = bytesutil.ToBytes48(deposits[i].Data.PublicKey)
		indices[i] = uint64(i)
	}

	chain := &mockChain.ChainService{
		State: bState, Root: genesisRoot[:], Genesis: time.Now(),
	}
	vs := &Server{
		BeaconDB:           db,
		HeadFetcher:        chain,
		GenesisTimeFetcher: chain,
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
	}

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
	}
	res, err := vs.GetDuties(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.CurrentEpochDuties), "Expected 1 assignment")
}

func TestGetDuties_MultipleKeys_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	genesis := testutil.NewBeaconBlock()
	depChainStart := uint64(64)
	testutil.ResetCache()
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	require.NoError(t, err, "Could not setup genesis bs")
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")

	pubKeys := make([][48]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = bytesutil.ToBytes48(deposits[i].Data.PublicKey)
		indices[i] = uint64(i)
	}

	chain := &mockChain.ChainService{
		State: bs, Root: genesisRoot[:], Genesis: time.Now(),
	}
	vs := &Server{
		BeaconDB:           db,
		HeadFetcher:        chain,
		GenesisTimeFetcher: chain,
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
	}

	pubkey0 := deposits[0].Data.PublicKey
	pubkey1 := deposits[1].Data.PublicKey

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{pubkey0, pubkey1},
	}
	res, err := vs.GetDuties(context.Background(), req)
	require.NoError(t, err, "Could not call epoch committee assignment")
	assert.Equal(t, 2, len(res.CurrentEpochDuties))
	assert.Equal(t, uint64(4), res.CurrentEpochDuties[0].AttesterSlot)
	assert.Equal(t, uint64(4), res.CurrentEpochDuties[1].AttesterSlot)
}

func TestGetDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetDuties(context.Background(), &ethpb.DutiesRequest{})
	assert.ErrorContains(t, "Syncing to latest head", err)
}

func TestStreamDuties_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_StreamDutiesServer(ctrl)
	assert.ErrorContains(t, "Syncing to latest head", vs.StreamDuties(&ethpb.DutiesRequest{}, mockStream))
}

func TestStreamDuties_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	genesis := testutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	require.NoError(t, err, "Could not setup genesis bs")
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	pubKeys := make([][]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
		indices[i] = uint64(i)
	}

	pubkeysAs48ByteType := make([][48]byte, len(pubKeys))
	for i, pk := range pubKeys {
		pubkeysAs48ByteType[i] = bytesutil.ToBytes48(pk)
	}

	ctx, cancel := context.WithCancel(context.Background())
	vs := &Server{
		Ctx:         ctx,
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mockChain.ChainService{
			Genesis: time.Now(),
		},
		StateNotifier: &mockChain.MockStateNotifier{},
	}

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
	}
	wantedRes, err := vs.duties(ctx, req)
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exitRoutine := make(chan bool)
	mockStream := mock.NewMockBeaconNodeValidator_StreamDutiesServer(ctrl)
	mockStream.EXPECT().Send(wantedRes).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()
	go func(tt *testing.T) {
		assert.ErrorContains(t, "context canceled", vs.StreamDuties(req, mockStream))
	}(t)
	<-exitRoutine
	cancel()
}

func TestStreamDuties_OK_ChainReorg(t *testing.T) {
	db, _ := dbutil.SetupDB(t)

	genesis := testutil.NewBeaconBlock()
	depChainStart := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(t, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(t, err)
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	require.NoError(t, err, "Could not setup genesis bs")
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")

	pubKeys := make([][]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = deposits[i].Data.PublicKey
		indices[i] = uint64(i)
	}

	pubkeysAs48ByteType := make([][48]byte, len(pubKeys))
	for i, pk := range pubKeys {
		pubkeysAs48ByteType[i] = bytesutil.ToBytes48(pk)
	}

	ctx, cancel := context.WithCancel(context.Background())
	vs := &Server{
		Ctx:         ctx,
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mockChain.ChainService{
			Genesis: time.Now(),
		},
		StateNotifier: &mockChain.MockStateNotifier{},
	}

	// Test the first validator in registry.
	req := &ethpb.DutiesRequest{
		PublicKeys: [][]byte{deposits[0].Data.PublicKey},
	}
	wantedRes, err := vs.duties(ctx, req)
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exitRoutine := make(chan bool)
	mockStream := mock.NewMockBeaconNodeValidator_StreamDutiesServer(ctrl)
	mockStream.EXPECT().Send(wantedRes).Return(nil)
	mockStream.EXPECT().Send(wantedRes).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()
	go func(tt *testing.T) {
		assert.ErrorContains(t, "context canceled", vs.StreamDuties(req, mockStream))
	}(t)
	// Fire a reorg event within the same epoch. This should NOT
	// trigger a recomputation not resending of duties over the stream.
	for sent := 0; sent == 0; {
		sent = vs.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Reorg,
			Data: &statefeed.ReorgData{OldSlot: 0, NewSlot: 0},
		})
	}
	// Fire a reorg event across epoch boundaries. This needs to trigger
	// a recomputation and resending of duties over the stream.
	for sent := 0; sent == 0; {
		sent = vs.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Reorg,
			Data: &statefeed.ReorgData{OldSlot: helpers.StartSlot(1), NewSlot: 0},
		})
	}
	<-exitRoutine
	cancel()
}

func TestAssignValidatorToSubnet(t *testing.T) {
	k := pubKey(3)

	assignValidatorToSubnet(k, ethpb.ValidatorStatus_ACTIVE)
	coms, ok, exp := cache.SubnetIDs.GetPersistentSubnets(k)
	require.Equal(t, true, ok, "No cache entry found for validator")
	assert.Equal(t, params.BeaconNetworkConfig().RandomSubnetsPerValidator, uint64(len(coms)))
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	totalTime := time.Duration(params.BeaconNetworkConfig().EpochsPerRandomSubnetSubscription) * epochDuration * time.Second
	receivedTime := exp.Round(time.Second).Sub(time.Now())
	if receivedTime < totalTime {
		t.Fatalf("Expiration time of %f was less than expected duration of %f ", receivedTime.Seconds(), totalTime.Seconds())
	}
}

func BenchmarkCommitteeAssignment(b *testing.B) {
	db, _ := dbutil.SetupDB(b)

	genesis := testutil.NewBeaconBlock()
	depChainStart := uint64(8192 * 2)
	deposits, _, err := testutil.DeterministicDepositsAndKeys(depChainStart)
	require.NoError(b, err)
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	require.NoError(b, err)
	bs, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	require.NoError(b, err, "Could not setup genesis bs")
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(b, err, "Could not get signing root")

	pubKeys := make([][48]byte, len(deposits))
	indices := make([]uint64, len(deposits))
	for i := 0; i < len(deposits); i++ {
		pubKeys[i] = bytesutil.ToBytes48(deposits[i].Data.PublicKey)
		indices[i] = uint64(i)
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
	}

	// Create request for all validators in the system.
	pks := make([][]byte, len(deposits))
	for i, deposit := range deposits {
		pks[i] = deposit.Data.PublicKey
	}
	req := &ethpb.DutiesRequest{
		PublicKeys: pks,
		Epoch:      0,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := vs.GetDuties(context.Background(), req)
		assert.NoError(b, err)
	}
}
