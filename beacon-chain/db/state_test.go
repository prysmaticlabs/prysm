package db

import (
	"bytes"
	"context"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func setupInitialDeposits(t testing.TB, numDeposits int) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		depositInput := &pb.DepositInput{
			Pubkey: priv.PublicKey().Marshal(),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func TestInitializeState_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(genesisTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}
	b, err := db.ChainHead()
	if err != nil {
		t.Fatalf("Failed to get chain head: %v", err)
	}
	if b.GetSlot() != params.BeaconConfig().GenesisSlot {
		t.Fatalf("Expected block height to equal 1. Got %d", b.GetSlot())
	}

	beaconState, err := db.State(ctx)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if beaconState == nil {
		t.Fatalf("Failed to retrieve state: %v", beaconState)
	}
	beaconStateEnc, err := proto.Marshal(beaconState)
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	statePrime, err := db.State(ctx)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	statePrimeEnc, err := proto.Marshal(statePrime)
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}

	if !bytes.Equal(beaconStateEnc, statePrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", beaconStateEnc, statePrimeEnc)
	}
}

func TestGenesisTime_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesisTime, err := db.GenesisTime(ctx)
	if err == nil {
		t.Fatal("Expected GenesisTime to fail")
	}

	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(uint64(genesisTime.Unix()), deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	time1, err := db.GenesisTime(ctx)
	if err != nil {
		t.Fatalf("GenesisTime failed on second attempt: %v", err)
	}
	time2, err := db.GenesisTime(ctx)
	if err != nil {
		t.Fatalf("GenesisTime failed on second attempt: %v", err)
	}

	if time1 != time2 {
		t.Fatalf("Expected %v and %v to be equal", time1, time2)
	}
}

func TestFinalizeState_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(genesisTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	state, err := db.State(context.Background())
	if err != nil {
		t.Fatalf("Failed to retrieve state: %v", err)
	}

	if err := db.SaveFinalizedState(state); err != nil {
		t.Fatalf("Unable to save finalized state")
	}

	fState, err := db.FinalizedState()
	if err != nil {
		t.Fatalf("Unable to retrieve finalized state")
	}

	if !proto.Equal(fState, state) {
		t.Error("Retrieved and saved finalized are unequal")
	}
}

func TestCurrentAndFinalizeState_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(genesisTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	state, err := db.State(context.Background())
	if err != nil {
		t.Fatalf("Failed to retrieve state: %v", err)
	}

	state.FinalizedEpoch = 10000
	state.BatchedBlockRootHash32S = [][]byte{[]byte("testing-prysm")}

	if err := db.SaveCurrentAndFinalizedState(state); err != nil {
		t.Fatalf("Unable to save state")
	}

	fState, err := db.FinalizedState()
	if err != nil {
		t.Fatalf("Unable to retrieve finalized state")
	}

	cState, err := db.State(context.Background())
	if err != nil {
		t.Fatalf("Unable to retrieve state")
	}

	if !proto.Equal(fState, state) {
		t.Error("Retrieved and saved finalized are unequal")
	}

	if !proto.Equal(cState, state) {
		t.Error("Retrieved and saved current are unequal")
	}
}

func BenchmarkState_ReadingFromCache(b *testing.B) {
	db := setupDB(b)
	defer teardownDB(b, db)
	ctx := context.Background()

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(b, 10)
	if err := db.InitializeState(genesisTime, deposits, &pb.Eth1Data{}); err != nil {
		b.Fatalf("Failed to initialize state: %v", err)
	}

	state, err := db.State(ctx)
	if err != nil {
		b.Fatalf("Could not read DV beacon state from DB: %v", err)
	}
	state.Slot++
	err = db.SaveState(state)
	if err != nil {
		b.Fatalf("Could not save beacon state to cache from DB: %v", err)
	}

	if db.currentState.Slot != params.BeaconConfig().GenesisSlot+1 {
		b.Fatal("cache should be prepared on state after saving to DB")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := db.State(ctx)
		if err != nil {
			b.Fatalf("Could not read beacon state from cache: %v", err)
		}
	}
}

func TestFinalizedState_NoneExists(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	wanted := "no finalized state saved"
	_, err := db.FinalizedState()
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestJustifiedState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	stateSlot := uint64(10)
	state := &pb.BeaconState{
		Slot: stateSlot,
	}

	if err := db.SaveJustifiedState(state); err != nil {
		t.Fatalf("could not save justified state: %v", err)
	}

	justifiedState, err := db.JustifiedState()
	if err != nil {
		t.Fatalf("could not get justified state: %v", err)
	}
	if justifiedState.Slot != stateSlot {
		t.Errorf("Saved state does not have the slot from which it was requested, wanted: %d, got: %d",
			stateSlot, justifiedState.Slot)
	}
}

func TestJustifiedState_NoneExists(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	wanted := "no justified state saved"
	_, err := db.JustifiedState()
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestFinalizedState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	stateSlot := uint64(10)
	state := &pb.BeaconState{
		Slot: stateSlot,
	}

	if err := db.SaveFinalizedState(state); err != nil {
		t.Fatalf("could not save finalized state: %v", err)
	}

	finalizedState, err := db.FinalizedState()
	if err != nil {
		t.Fatalf("could not get finalized state: %v", err)
	}
	if finalizedState.Slot != stateSlot {
		t.Errorf("Saved state does not have the slot from which it was requested, wanted: %d, got: %d",
			stateSlot, finalizedState.Slot)
	}
}

func TestHistoricalState_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	tests := []struct {
		state *pb.BeaconState
	}{
		{
			state: &pb.BeaconState{
				Slot:           66,
				FinalizedEpoch: 1,
			},
		},
		{
			state: &pb.BeaconState{
				Slot:           72,
				FinalizedEpoch: 1,
			},
		},
		{
			state: &pb.BeaconState{
				Slot:           96,
				FinalizedEpoch: 1,
			},
		},
		{
			state: &pb.BeaconState{
				Slot:           130,
				FinalizedEpoch: 2,
			},
		},
		{
			state: &pb.BeaconState{
				Slot:           300,
				FinalizedEpoch: 4,
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveFinalizedState(tt.state); err != nil {
			t.Fatalf("could not save finalized state: %v", err)
		}
		if err := db.SaveHistoricalState(tt.state); err != nil {
			t.Fatalf("could not save historical state: %v", err)
		}

		retState, err := db.HistoricalStateFromSlot(tt.state.Slot)
		if err != nil {
			t.Fatalf("Unable to retrieve state %v", err)
		}

		if !proto.Equal(tt.state, retState) {
			t.Errorf("Saved and retrieved states are not equal got\n %v but wanted\n %v", proto.MarshalTextString(retState), proto.MarshalTextString(tt.state))
		}
	}
}

func TestHistoricalState_Pruning(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	epochSize := params.BeaconConfig().SlotsPerEpoch
	slotGen := func(slot uint64) uint64 {
		return params.BeaconConfig().GenesisSlot + slot
	}

	tests := []struct {
		finalizedState *pb.BeaconState
		histState1     *pb.BeaconState
		histState2     *pb.BeaconState
	}{
		{
			finalizedState: &pb.BeaconState{
				Slot:           slotGen(2 * epochSize),
				FinalizedEpoch: 2,
			},
			histState1: &pb.BeaconState{
				Slot:           slotGen(1 * epochSize),
				FinalizedEpoch: 1,
			},
			histState2: &pb.BeaconState{
				Slot:           slotGen(4 * epochSize),
				FinalizedEpoch: 2,
			},
		},
		{
			finalizedState: &pb.BeaconState{
				Slot:           slotGen(4 * epochSize),
				FinalizedEpoch: 4,
			},
			histState1: &pb.BeaconState{
				Slot:           slotGen(2 * epochSize),
				FinalizedEpoch: 2,
			},
			histState2: &pb.BeaconState{
				Slot:           slotGen(5 * epochSize),
				FinalizedEpoch: 4,
			},
		},
		{
			finalizedState: &pb.BeaconState{
				Slot:           slotGen(12 * epochSize),
				FinalizedEpoch: 12,
			},
			histState1: &pb.BeaconState{
				Slot:           slotGen(6 * epochSize),
				FinalizedEpoch: 6,
			},
			histState2: &pb.BeaconState{
				Slot:           slotGen(14 * epochSize),
				FinalizedEpoch: 14,
			},
		},
		{
			finalizedState: &pb.BeaconState{
				Slot:           slotGen(100 * epochSize),
				FinalizedEpoch: 100,
			},
			histState1: &pb.BeaconState{
				Slot:           slotGen(12 * epochSize),
				FinalizedEpoch: 12,
			},
			histState2: &pb.BeaconState{
				Slot:           slotGen(103 * epochSize),
				FinalizedEpoch: 103,
			},
		},
		{
			finalizedState: &pb.BeaconState{
				Slot:           slotGen(500 * epochSize),
				FinalizedEpoch: 500,
			},
			histState1: &pb.BeaconState{
				Slot:           slotGen(100 * epochSize),
				FinalizedEpoch: 100,
			},
			histState2: &pb.BeaconState{
				Slot:           slotGen(600 * epochSize),
				FinalizedEpoch: 600,
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveFinalizedState(tt.finalizedState); err != nil {
			t.Fatalf("could not save finalized state: %v", err)
		}
		if err := db.SaveHistoricalState(tt.histState1); err != nil {
			t.Fatalf("could not save historical state: %v", err)
		}
		if err := db.SaveHistoricalState(tt.histState2); err != nil {
			t.Fatalf("could not save historical state: %v", err)
		}

		if err := db.deleteHistoricalStates(tt.finalizedState.Slot); err != nil {
			t.Fatalf("Could not delete historical states %v", err)
		}

		retState, err := db.HistoricalStateFromSlot(tt.histState1.Slot)
		if err != nil {
			t.Fatalf("Unable to retrieve state %v", err)
		}

		if proto.Equal(tt.histState1, retState) {
			t.Errorf("Saved and retrieved states are equal when they aren't supposed to be for slot %d", retState.Slot)
		}

		retState, err = db.HistoricalStateFromSlot(tt.histState2.Slot)
		if err != nil {
			t.Fatalf("Unable to retrieve state %v", err)
		}

		if !proto.Equal(tt.histState2, retState) {
			t.Errorf("Saved and retrieved states are not equal when they supposed to be for slot %d", tt.histState2.Slot-params.BeaconConfig().GenesisSlot)
		}

	}
}
