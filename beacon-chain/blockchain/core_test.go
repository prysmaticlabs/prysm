package blockchain

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/geth-sharding/beacon-chain/database"
	"github.com/prysmaticlabs/geth-sharding/beacon-chain/types"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestNewBeaconChain(t *testing.T) {
	hook := logTest.NewGlobal()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	db, err := database.NewBeaconDB(context.Background(), tmp, "beacontest", false)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	db.Start()
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	msg := hook.LastEntry().Message
	want := "No chainstate found on disk, initializing beacon from genesis"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	hook.Reset()
	active, crystallized := types.NewGenesisStates()
	if !reflect.DeepEqual(beaconChain.ActiveState(), active) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconChain.ActiveState(), active)
	}
	if !reflect.DeepEqual(beaconChain.CrystallizedState(), crystallized) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconChain.CrystallizedState(), crystallized)
	}
}

func TestMutateActiveState(t *testing.T) {
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	db, err := database.NewBeaconDB(context.Background(), tmp, "beacontest2", false)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	db.Start()
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	randao := common.BytesToHash([]byte("hello"))
	active := &types.ActiveState{
		Height: 100,
		Randao: randao,
	}
	if err := beaconChain.MutateActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.ActiveState, active) {
		t.Errorf("active state was not updated. wanted %v, got %v", active, beaconChain.state.ActiveState)
	}

	// Initializing a new beacon chain should deserialize persisted state from disk.
	newBeaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}
	// The active state should still be the one we mutated and persited earlier.
	if active.Height != newBeaconChain.state.ActiveState.Height {
		t.Errorf("active state height incorrect. wanted %v, got %v", active.Height, newBeaconChain.state.ActiveState.Height)
	}
	if active.Randao.Hex() != newBeaconChain.state.ActiveState.Randao.Hex() {
		t.Errorf("active state randao incorrect. wanted %v, got %v", active.Randao.Hex(), newBeaconChain.state.ActiveState.Randao.Hex())
	}
}

func TestMutateCrystallizedState(t *testing.T) {
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	db, err := database.NewBeaconDB(context.Background(), tmp, "beacontest3", false)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	db.Start()
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	currentCheckpoint := common.BytesToHash([]byte("checkpoint"))
	crystallized := &types.CrystallizedState{
		Dynasty:           3,
		CurrentCheckpoint: currentCheckpoint,
	}
	if err := beaconChain.MutateCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.CrystallizedState, crystallized) {
		t.Errorf("crystallized state was not updated. wanted %v, got %v", crystallized, beaconChain.state.CrystallizedState)
	}

	// Initializing a new beacon chain should deserialize persisted state from disk.
	newBeaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}
	// The crystallized state should still be the one we mutated and persited earlier.
	if crystallized.Dynasty != newBeaconChain.state.CrystallizedState.Dynasty {
		t.Errorf("crystallized state dynasty incorrect. wanted %v, got %v", crystallized.Dynasty, newBeaconChain.state.CrystallizedState.Dynasty)
	}
	if crystallized.CurrentCheckpoint.Hex() != newBeaconChain.state.CrystallizedState.CurrentCheckpoint.Hex() {
		t.Errorf("crystallized state current checkpoint incorrect. wanted %v, got %v", crystallized.CurrentCheckpoint.Hex(), newBeaconChain.state.CrystallizedState.CurrentCheckpoint.Hex())
	}
}
