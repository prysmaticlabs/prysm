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

	hook.Reset()
	aState := types.NewGenesisActiveState()
	aState2 := types.NewGenesisActiveState()

	if !reflect.DeepEqual(aState, aState2) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", aState, aState2)
	}
}
