package helpers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestRandaoMix_OK(t *testing.T) {
	randaoMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	state := &pb.BeaconState{LatestRandaoMixes: randaoMixes}
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
			randaoMix: randaoMixes[99999%params.BeaconConfig().LatestRandaoMixesLength],
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch + 1) * params.BeaconConfig().SlotsPerEpoch
		mix, err := RandaoMix(state, test.epoch)
		if err != nil {
			t.Fatalf("Could not get randao mix: %v", err)
		}
		if !bytes.Equal(test.randaoMix, mix) {
			t.Errorf("Incorrect randao mix. Wanted: %#x, got: %#x",
				test.randaoMix, mix)
		}
	}
}

func TestRandaoMix_OutOfBound(t *testing.T) {
	wanted := fmt.Sprintf(
		"input randaoMix epoch %d out of bounds: %d <= epoch < %d",
		100, 0, 0,
	)
	if _, err := RandaoMix(&pb.BeaconState{}, 100); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestActiveIndexRoot_OK(t *testing.T) {
	activeIndexRoots := make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		activeIndexRoots[i] = intInBytes
	}
	state := &pb.BeaconState{LatestIndexRootHash32S: activeIndexRoots}
	tests := []struct {
		epoch     uint64
		indexRoot []byte
	}{
		{
			epoch:     34,
			indexRoot: activeIndexRoots[34],
		},
		{
			epoch:     3444,
			indexRoot: activeIndexRoots[3444],
		},
		{
			epoch:     999999,
			indexRoot: activeIndexRoots[999999%params.BeaconConfig().LatestActiveIndexRootsLength],
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch + 1) * params.BeaconConfig().SlotsPerEpoch
		indexRoot, err := ActiveIndexRoot(state, test.epoch)
		if err != nil {
			t.Fatalf("Could not get index root: %v", err)
		}
		if !bytes.Equal(test.indexRoot, indexRoot) {
			t.Errorf("Incorrect index root. Wanted: %#x, got: %#x",
				test.indexRoot, indexRoot)
		}
	}
}

func TestActiveIndexRoot_OutOfBound(t *testing.T) {
	wanted := fmt.Sprintf(
		"input indexRoot epoch %d out of bounds: %d <= epoch < %d",
		100, 0, 0,
	)
	if _, err := ActiveIndexRoot(&pb.BeaconState{}, 100); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestGenerateSeed_OutOfBound(t *testing.T) {
	wanted := fmt.Sprintf(
		"input randaoMix epoch %d out of bounds: %d <= epoch < %d",
		100-params.BeaconConfig().MinSeedLookahead, 0, 0,
	)
	if _, err := GenerateSeed(&pb.BeaconState{}, 100); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestGenerateSeed_OK(t *testing.T) {
	activeIndexRoots := make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		activeIndexRoots[i] = intInBytes
	}
	randaoMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	slot := 10 * params.BeaconConfig().MinSeedLookahead * params.BeaconConfig().SlotsPerEpoch
	state := &pb.BeaconState{
		LatestIndexRootHash32S: activeIndexRoots,
		LatestRandaoMixes:      randaoMixes,
		Slot:                   slot}

	got, err := GenerateSeed(state, 10)
	if err != nil {
		t.Fatalf("Could not generate seed: %v", err)
	}
	wanted := [32]byte{242, 35, 140, 234, 201, 60, 23, 152, 187, 111, 227, 129, 108, 254, 108, 63,
		192, 111, 114, 225, 0, 159, 145, 106, 133, 217, 25, 25, 251, 58, 175, 168}
	if got != wanted {
		t.Errorf("Incorrect generated seeds. Got: %v, wanted: %v",
			got, wanted)
	}
}
