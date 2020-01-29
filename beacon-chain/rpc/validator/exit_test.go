package validator

import (
	"context"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	blk "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSub(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	deposits, _, _ := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	block := blk.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	// Set genesis time to be 100 epochs ago.
	genesisTime := time.Now().Add(time.Duration(-100*int64(params.BeaconConfig().SecondsPerSlot*params.BeaconConfig().SlotsPerEpoch)) * time.Second)
	mockChainService := &mockChain.ChainService{State: beaconState, Root: genesisRoot[:], Genesis: genesisTime}
	server := &Server{
		BeaconDB:          db,
		HeadFetcher:       mockChainService,
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		GenesisTime:       genesisTime,
		StateNotifier:     mockChainService.StateNotifier(),
		OperationNotifier: mockChainService.OperationNotifier(),
		ExitPool:          voluntaryexits.NewPool(),
		P2P:               mockp2p.NewTestP2P(t),
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
		Signature: []byte{0xb3, 0xe1, 0x9d, 0xc6, 0x7c, 0x78, 0x6c, 0xcf, 0x33, 0x1d, 0xb9, 0x6f, 0x59, 0x64, 0x44, 0xe1, 0x29, 0xd0, 0x87, 0x03, 0x26, 0x6e, 0x49, 0x1c, 0x05, 0xae, 0x16, 0x7b, 0x04, 0x0f, 0x3f, 0xf8, 0x82, 0x77, 0x60, 0xfc, 0xcf, 0x2f, 0x59, 0xc7, 0x40, 0x0b, 0x2c, 0xa9, 0x23, 0x8a, 0x6c, 0x8d, 0x01, 0x21, 0x5e, 0xa8, 0xac, 0x36, 0x70, 0x31, 0xb0, 0xe1, 0xa8, 0xb8, 0x8f, 0x93, 0x8c, 0x1c, 0xa2, 0x86, 0xe7, 0x22, 0x00, 0x6a, 0x7d, 0x36, 0xc0, 0x2b, 0x86, 0x2c, 0xf5, 0xf9, 0x10, 0xb9, 0xf2, 0xbd, 0x5e, 0xa6, 0x5f, 0x12, 0x86, 0x43, 0x20, 0x4d, 0xa2, 0x9d, 0x8b, 0xe6, 0x6f, 0x09},
	}

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
				data := event.Data.(*opfeed.ExitReceivedData)
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
