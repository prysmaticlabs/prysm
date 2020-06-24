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
)

func TestSub(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	testutil.ResetCache()
	deposits, keys, err := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

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
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainVoluntaryExit, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	sigRoot, err := helpers.ComputeSigningRoot(req.Exit, domain)
	if err != nil {
		t.Fatalf("Could not compute signing root: %v", err)
	}
	req.Signature = keys[0].Sign(sigRoot[:]).Marshal()

	_, err = server.ProposeExit(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Ensure the state notification was broadcast.
	notificationFound := false
	for !notificationFound {
		select {
		case event := <-opChannel:
			if event.Type == opfeed.ExitReceived {
				notificationFound = true
				data, ok := event.Data.(*opfeed.ExitReceivedData)
				if !ok {
					t.Error("Entity is not of type *opfeed.ExitReceivedData")
				}
				if epoch != data.Exit.Exit.Epoch {
					t.Errorf("Unexpected state feed epoch: expected %v, found %v", epoch, data.Exit.Exit.Epoch)
				}
				if validatorIndex != data.Exit.Exit.ValidatorIndex {
					t.Errorf("Unexpected state feed validator index: expected %v, found %v", validatorIndex, data.Exit.Exit.ValidatorIndex)
				}
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
	if err != nil {
		t.Fatal(err)
	}
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	block := testutil.NewBeaconBlock()
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

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
	if err == nil {
		t.Fatal("Expected error for no exit existing")
	}

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
	if err == nil {
		t.Fatal("Expected error for no signature exists")
	}
	req.Signature = bytesutil.FromBytes48([48]byte{})

	_, err = server.ProposeExit(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for invalid signature length")
	}

	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainVoluntaryExit, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	sigRoot, err := helpers.ComputeSigningRoot(req.Exit, domain)
	if err != nil {
		t.Fatalf("Could not compute signing root: %v", err)
	}
	req.Signature = keys[0].Sign(sigRoot[:]).Marshal()

	_, err = server.ProposeExit(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
