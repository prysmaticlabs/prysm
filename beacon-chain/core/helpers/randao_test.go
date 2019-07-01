package helpers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"runtime"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestRandaoMix_OK(t *testing.T) {
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	state := &pb.BeaconState{RandaoMixes: randaoMixes}
	tests := []struct {
		epoch     uint64
		randaoMix []byte
	}{
		{
			epoch:     10,
			randaoMix: randaoMixes[10],
		},
		{
			epoch:     2344,
			randaoMix: randaoMixes[2344],
		},
		{
			epoch:     99999,
			randaoMix: randaoMixes[99999%params.BeaconConfig().EpochsPerHistoricalVector],
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch + 1) * params.BeaconConfig().SlotsPerEpoch
		mix := RandaoMix(state, test.epoch)
		if !bytes.Equal(test.randaoMix, mix) {
			t.Errorf("Incorrect randao mix. Wanted: %#x, got: %#x",
				test.randaoMix, mix)
		}
	}
}

func TestRandaoMix_CopyOK(t *testing.T) {
	ClearAllCaches()
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	state := &pb.BeaconState{RandaoMixes: randaoMixes}
	tests := []struct {
		epoch     uint64
		randaoMix []byte
	}{
		{
			epoch:     10,
			randaoMix: randaoMixes[10],
		},
		{
			epoch:     2344,
			randaoMix: randaoMixes[2344],
		},
		{
			epoch:     99999,
			randaoMix: randaoMixes[99999%params.BeaconConfig().EpochsPerHistoricalVector],
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch + 1) * params.BeaconConfig().SlotsPerEpoch
		mix := RandaoMix(state, test.epoch)
		randaoMap := make(map[string]bool)
		for _, elem := range mix {
			randaoMap[fmt.Sprintf("%v", &elem)] = true
		}
		for _, mx := range randaoMixes {
			for _, val := range mx {
				if randaoMap[fmt.Sprintf("%v", &val)] {
					t.Fatalf("two distinct slices still have elements referenced by the same address: %v", &val)
				}
			}
		}
		runtime.GC()
	}
}

func TestActiveIndexRoot_OK(t *testing.T) {

	activeIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		activeIndexRoots[i] = intInBytes
	}
	state := &pb.BeaconState{ActiveIndexRoots: activeIndexRoots}
	tests := []struct {
		epoch uint64
	}{
		{
			epoch: 34,
		},
		{
			epoch: 3444,
		},
		{
			epoch: 999999,
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch) * params.BeaconConfig().SlotsPerEpoch
		for i := 0; i <= int(params.BeaconConfig().ActivationExitDelay); i++ {
			indexRoot := ActiveIndexRoot(state, test.epoch+uint64(i))

			if !bytes.Equal(activeIndexRoots[(test.epoch+uint64(i))%params.BeaconConfig().EpochsPerHistoricalVector], indexRoot) {
				t.Errorf("Incorrect index root. Wanted: %#x, got: %#x",
					activeIndexRoots[(test.epoch+uint64(i))%params.BeaconConfig().EpochsPerHistoricalVector], indexRoot)
			}
		}

	}
}

func TestActiveIndexRoot_CopyOK(t *testing.T) {
	ClearAllCaches()
	runtime.GC()

	activeIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		activeIndexRoots[i] = intInBytes
	}
	state := &pb.BeaconState{ActiveIndexRoots: activeIndexRoots}
	tests := []struct {
		epoch uint64
	}{
		{
			epoch: 34,
		},
		{
			epoch: 3444,
		},
		{
			epoch: 999999,
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch) * params.BeaconConfig().SlotsPerEpoch
		indexRoot := ActiveIndexRoot(state, test.epoch)
		rootMap := make(map[string]bool)
		for _, elem := range indexRoot {
			rootMap[fmt.Sprintf("%v", &elem)] = true
		}
		for _, root := range activeIndexRoots {
			for _, val := range root {
				if rootMap[fmt.Sprintf("%v", &val)] {
					t.Fatalf("two distinct slices still have elements referenced by the same address: %v", &val)
				}
			}
		}
		runtime.GC()
	}
}

func TestGenerateSeed_OK(t *testing.T) {
	ClearAllCaches()

	activeIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		activeIndexRoots[i] = intInBytes
	}
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	slot := 10 * params.BeaconConfig().MinSeedLookahead * params.BeaconConfig().SlotsPerEpoch
	state := &pb.BeaconState{
		ActiveIndexRoots: activeIndexRoots,
		RandaoMixes:      randaoMixes,
		Slot:             slot}

	got, err := GenerateSeed(state, 10)
	if err != nil {
		t.Fatal(err)
	}

	wanted := [32]byte{239, 112, 63, 86, 124, 180, 155, 181, 91, 67, 231, 178,
		94, 149, 243, 101, 176, 169, 153, 35, 37, 19, 115, 154, 6, 102, 125, 91, 81, 153, 186, 84}
	if got != wanted {
		t.Errorf("Incorrect generated seeds. Got: %v, wanted: %v",
			got, wanted)
	}
}
