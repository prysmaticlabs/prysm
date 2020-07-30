package validator

import (
	"context"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSub(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	testutil.ResetCache()
	deposits, keys, err := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	require.NoError(t, err)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	// Set genesis time to be 100 epochs ago.
	genesisTime := time.Now().Add(time.Duration(-100*int64(params.BeaconConfig().SecondsPerSlot*params.BeaconConfig().SlotsPerEpoch)) * time.Second)
	mockChainService := &mockChain.ChainService{State: beaconState, Root: genesisRoot[:], Genesis: genesisTime}
	server := &Server{
		BeaconDB:           db,
		HeadFetcher:        mockChainService,
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: mockChainService,
		StateNotifier:      mockChainService.StateNotifier(),
		OperationNotifier:  mockChainService.OperationNotifier(),
		ExitPool:           voluntaryexits.NewPool(),
		P2P:                mockp2p.NewTestP2P(t),
	}

	// Subscribe to operation notifications.
	opChannel := make(chan *feed.Event, 1024)
	opSub := server.OperationNotifier.OperationFeed().Subscribe(opChannel)
	defer opSub.Unsubscribe()

	// Send the request, expect a result on the state feed.
	epoch := uint64(2048)
	validatorIndex := uint64(0)
	req := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			Epoch:          epoch,
			ValidatorIndex: validatorIndex,
		},
	}
	req.Signature, err = helpers.ComputeDomainAndSign(beaconState, epoch, req.Exit, params.BeaconConfig().DomainVoluntaryExit, keys[0])
	require.NoError(t, err)

	_, err = server.ProposeExit(context.Background(), req)
	require.NoError(t, err)

	// Ensure the state notification was broadcast.
	notificationFound := false
	for !notificationFound {
		select {
		case event := <-opChannel:
			if event.Type == opfeed.ExitReceived {
				notificationFound = true
				data, ok := event.Data.(*opfeed.ExitReceivedData)
				assert.Equal(t, true, ok, "Entity is not of type *opfeed.ExitReceivedData")
				assert.Equal(t, epoch, data.Exit.Exit.Epoch, "Unexpected state feed epoch")
				assert.Equal(t, validatorIndex, data.Exit.Exit.ValidatorIndex, "Unexpected state feed validator index")
			}
		case <-opSub.Err():
			t.Error("Subscription to state notifier failed")
			return
		}
	}
}

func TestProposeExit_NoPanic(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	testutil.ResetCache()
	deposits, keys, err := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	require.NoError(t, err)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, block), "Could not save genesis block")
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err, "Could not get signing root")

	// Set genesis time to be 100 epochs ago.
	genesisTime := time.Now().Add(time.Duration(-100*int64(params.BeaconConfig().SecondsPerSlot*params.BeaconConfig().SlotsPerEpoch)) * time.Second)
	mockChainService := &mockChain.ChainService{State: beaconState, Root: genesisRoot[:], Genesis: genesisTime}
	server := &Server{
		BeaconDB:           db,
		HeadFetcher:        mockChainService,
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: mockChainService,
		StateNotifier:      mockChainService.StateNotifier(),
		OperationNotifier:  mockChainService.OperationNotifier(),
		ExitPool:           voluntaryexits.NewPool(),
		P2P:                mockp2p.NewTestP2P(t),
	}

	// Subscribe to operation notifications.
	opChannel := make(chan *feed.Event, 1024)
	opSub := server.OperationNotifier.OperationFeed().Subscribe(opChannel)
	defer opSub.Unsubscribe()

	req := &ethpb.SignedVoluntaryExit{}
	_, err = server.ProposeExit(context.Background(), req)
	require.ErrorContains(t, "voluntary exit does not exist", err, "Expected error for no exit existing")

	// Send the request, expect a result on the state feed.
	epoch := uint64(2048)
	validatorIndex := uint64(0)
	req = &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			Epoch:          epoch,
			ValidatorIndex: validatorIndex,
		},
	}

	_, err = server.ProposeExit(context.Background(), req)
	require.ErrorContains(t, "invalid signature provided", err, "Expected error for no signature exists")
	req.Signature = bytesutil.FromBytes48([48]byte{})

	_, err = server.ProposeExit(context.Background(), req)
	require.ErrorContains(t, "invalid signature provided", err, "Expected error for invalid signature length")
	req.Signature, err = helpers.ComputeDomainAndSign(beaconState, epoch, req.Exit, params.BeaconConfig().DomainVoluntaryExit, keys[0])
	require.NoError(t, err)
	_, err = server.ProposeExit(context.Background(), req)
	require.NoError(t, err)
}
