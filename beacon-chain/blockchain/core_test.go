package blockchain

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/testutils"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type faultyDB struct{}

func (f *faultyDB) Get(k []byte) ([]byte, error) {
	return []byte{}, nil
}

func (f *faultyDB) Has(k []byte) (bool, error) {
	return true, nil
}

func (f *faultyDB) Put(k []byte, v []byte) error {
	return nil
}

func (f *faultyDB) Delete(k []byte) error {
	return nil
}

func (f *faultyDB) Close() {}

func (f *faultyDB) NewBatch() ethdb.Batch {
	return nil
}

func startBeaconChain(t *testing.T) *BeaconChain {
	beaconChain, err := NewBeaconChain(testutils.SetupDB(t))
	if err != nil {
		t.Fatalf("failed to instantiate beacon chain: %v", err)
	}

	return beaconChain
}

func TestNewBeaconChain(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconChain := startBeaconChain(t)

	msg := hook.LastEntry().Message
	want := "No state found on disk, initializing genesis block and state"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	hook.Reset()
	aState := types.NewGenesisActiveState()
	aState2 := types.NewGenesisActiveState()
	if !reflect.DeepEqual(beaconChain.ActiveState(), aState2) {
		t.Errorf("there's something wrong here")
	}
	// cState, err := types.NewGenesisCrystallizedState()
	// if err != nil {
	// 	t.Errorf("Creating new genesis state failed %v", err)
	// }
	// if _, err := types.NewGenesisBlock(); err != nil {
	// 	t.Errorf("Creating a new genesis block failed %v", err)
	// }

	if !reflect.DeepEqual(beaconChain.ActiveState(), aState) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconChain.ActiveState(), aState)
	}

	// if !reflect.DeepEqual(beaconChain.CrystallizedState(), cState) {
	// 	t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconChain.CrystallizedState(), cState)
	// }
	// if _, err := beaconChain.GenesisBlock(); err != nil {
	// 	t.Errorf("Getting new beaconchain genesis failed: %v", err)
	// }
}
