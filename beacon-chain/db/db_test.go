package db

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func startInMemoryBeaconDB(t *testing.T) *BeaconDB {
	config := Config{Path: "", Name: "", InMemory: true}
	db, err := NewDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}

	return db
}

func TestNewDB(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	msg := hook.LastEntry().Message
	want := "No chainstate found on disk, initializing beacon from genesis"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	hook.Reset()
	aState := types.NewGenesisActiveState()
	cState, err := types.NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Errorf("Creating new genesis state failed %v", err)
	}

	if !proto.Equal(beaconDB.GetActiveState().Proto(), aState.Proto()) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconDB.GetActiveState(), aState)
	}

	if !proto.Equal(beaconDB.GetCrystallizedState().Proto(), cState.Proto()) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconDB.GetCrystallizedState(), cState)
	}
}
