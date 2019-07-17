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
	conf := params.BeaconConfig()
	conf.EpochsPerHistoricalVector = 100
	params.OverrideBeaconConfig(conf)
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
	}
	for _, test := range tests {
		state.Slot = (test.epoch) * params.BeaconConfig().SlotsPerEpoch
		indexRoot := ActiveIndexRoot(state, test.epoch)
		uniqueNumber := params.BeaconConfig().EpochsPerHistoricalVector + 1000
		binary.LittleEndian.PutUint64(indexRoot, uniqueNumber)

		for _, root := range activeIndexRoots {
			rootNum := bytesutil.FromBytes8(root)
			if rootNum == uniqueNumber {
				t.Fatalf("two distinct slices which have different representations in memory still contain"+
					"the same value: %d", rootNum)
			}
		}
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

	got, err := Seed(state, 10)
	if err != nil {
		t.Fatal(err)
	}

	wanted := [32]byte{141, 205, 112, 76, 60, 173, 127, 10, 1, 214, 151, 41, 69, 40, 108, 88, 247,
		210, 88, 5, 150, 112, 64, 93, 208, 110, 194, 137, 234, 180, 40, 245}
	if got != wanted {
		t.Errorf("Incorrect generated seeds. Got: %v, wanted: %v",
			got, wanted)
	}
}
