package helpers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestRandaoMix_Ok(t *testing.T) {
	randaoMixes := make([][]byte, config.LatestRandaoMixesLength)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.BigEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	state := &pb.BeaconState{LatestRandaoMixesHash32S: randaoMixes}
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
			randaoMix: randaoMixes[99999%config.LatestRandaoMixesLength],
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch + 1) * config.EpochLength
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
		"input randaoMix epoch %d out of bounds: %d <= slot < %d",
		100, 0, 0,
	)
	if _, err := RandaoMix(&pb.BeaconState{}, 100); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestActiveIndexRoot_Ok(t *testing.T) {
	activeIndexRoots := make([][]byte, config.LatestIndexRootsLength)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.BigEndian.PutUint64(intInBytes, uint64(i))
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
			indexRoot: activeIndexRoots[999999%config.LatestIndexRootsLength],
		},
	}
	for _, test := range tests {
		state.Slot = (test.epoch + 1) * config.EpochLength
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
		"input indexRoot epoch %d out of bounds: %d <= slot < %d",
		100, 0, 0,
	)
	if _, err := ActiveIndexRoot(&pb.BeaconState{}, 100); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestGenerateSeed_OutOfBound(t *testing.T) {
	wanted := fmt.Sprintf(
		"input randaoMix epoch %d out of bounds: %d <= slot < %d",
		100-config.SeedLookahead, 0, 0,
	)
	if _, err := GenerateSeed(&pb.BeaconState{}, 100); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestGenerateSeed_Ok(t *testing.T) {
	activeIndexRoots := make([][]byte, config.LatestIndexRootsLength)
	for i := 0; i < len(activeIndexRoots); i++ {
		intInBytes := make([]byte, 32)
		binary.BigEndian.PutUint64(intInBytes, uint64(i))
		activeIndexRoots[i] = intInBytes
	}
	randaoMixes := make([][]byte, config.LatestRandaoMixesLength)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.BigEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	slot := 10 * config.SeedLookahead * config.EpochLength
	state := &pb.BeaconState{
		LatestIndexRootHash32S:   activeIndexRoots,
		LatestRandaoMixesHash32S: randaoMixes,
		Slot:                     slot}

	got, err := GenerateSeed(state, 10*config.EpochLength)
	if err != nil {
		t.Fatalf("Could not generate seed: %v", err)
	}
	wanted := [32]byte{180, 101, 143, 188, 76, 11, 89, 58, 240, 23, 26, 115, 123, 117, 241, 87,
		150, 254, 168, 69, 44, 64, 142, 223, 32, 182, 2, 247, 191, 208, 189, 152}
	if got != wanted {
		t.Errorf("Incorrect generated seeds. Got: %v, wanted: %v",
			got, wanted)
	}
}
