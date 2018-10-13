package db

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveActiveState(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	data := &pb.ActiveState{
		PendingAttestations: []*pb.AggregatedAttestation{
			{Slot: 0, ShardBlockHash: []byte{1}}, {Slot: 1, ShardBlockHash: []byte{2}},
		},
		RecentBlockHashes: [][]byte{
			{'A'}, {'B'}, {'C'}, {'D'},
		},
	}
	active := types.NewActiveState(data, make(map[[32]byte]*types.VoteCache))

	if err := beaconDB.SaveActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	if !reflect.DeepEqual(beaconDB.GetActiveState(), active) {
		t.Errorf("active state was not updated. wanted %v, got %v", active, beaconDB.state.aState)
	}
}

func TestSaveCrystallizedState(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	data := &pb.CrystallizedState{
		ValidatorSetChangeSlot: 3,
	}
	crystallized := types.NewCrystallizedState(data)

	if err := beaconDB.SaveCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}
	if !reflect.DeepEqual(beaconDB.state.cState, crystallized) {
		t.Errorf("crystallized state was not updated. wanted %v, got %v", crystallized, beaconDB.state.cState)
	}
}
