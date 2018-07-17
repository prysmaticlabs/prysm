package blockchain

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/geth-sharding/beacon-chain/types"
	sharedDB "github.com/prysmaticlabs/geth-sharding/shared/database"
)

func TestNewBeaconChain(t *testing.T) {
	db := sharedDB.NewKVStore()
	beaconChain, err := NewBeaconChain(db)
	if err != nil {
		t.Fatalf("unable to setup beacon chain")
	}
	active, crystallized := types.NewGenesisStates()
	if !reflect.DeepEqual(beaconChain.activeState, active) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconChain.activeState, active)
	}
	if !reflect.DeepEqual(beaconChain.crystallizedState, crystallized) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconChain.crystallizedState, crystallized)
	}
}
