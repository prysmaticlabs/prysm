package state

import (
	"sync"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestBeaconState_SlotDataRace(t *testing.T) {
	headState, _ := InitializeFromProto(&pb.BeaconState{Slot: 1})

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		headState.SetSlot(uint64(0))
		wg.Done()
	}()
	go func() {
		headState.Slot()
		wg.Done()
	}()

	wg.Wait()
}
