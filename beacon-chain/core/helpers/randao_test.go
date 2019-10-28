package helpers

import (
	"bytes"
	"encoding/binary"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		uniqueNumber := params.BeaconConfig().EpochsPerHistoricalVector + 1000
		binary.LittleEndian.PutUint64(mix, uniqueNumber)

		for _, mx := range randaoMixes {
			mxNum := bytesutil.FromBytes8(mx)
			if mxNum == uniqueNumber {
				t.Fatalf("two distinct slices which have different representations in memory still contain"+
					"the same value: %d", mxNum)
			}
		}
	}
}

func TestGenerateSeed_OK(t *testing.T) {
	ClearAllCaches()
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	slot := 10 * params.BeaconConfig().MinSeedLookahead * params.BeaconConfig().SlotsPerEpoch
	state := &pb.BeaconState{
		RandaoMixes: randaoMixes,
		Slot:        slot}

	got, err := Seed(state, 10)
	if err != nil {
		t.Fatal(err)
	}

	wanted := [32]byte{144, 191, 18, 102, 76, 239, 2, 72, 255, 140, 52, 39, 113, 216, 101, 58, 241, 18, 179, 159, 45, 50, 57, 170, 9, 135, 91, 221, 97, 236, 157, 28}
	if got != wanted {
		t.Errorf("Incorrect generated seeds. Got: %v, wanted: %v",
			got, wanted)
	}
}
