package helpers

import (
	"encoding/binary"
	"testing"

	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRandaoMix_OK(t *testing.T) {
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{RandaoMixes: randaoMixes})
	require.NoError(t, err)
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
		require.NoError(t, state.SetSlot((test.epoch+1)*params.BeaconConfig().SlotsPerEpoch))
		mix, err := RandaoMix(state, test.epoch)
		require.NoError(t, err)
		assert.DeepEqual(t, test.randaoMix, mix, "Incorrect randao mix")
	}
}

func TestRandaoMix_CopyOK(t *testing.T) {
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{RandaoMixes: randaoMixes})
	require.NoError(t, err)
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
		require.NoError(t, state.SetSlot((test.epoch+1)*params.BeaconConfig().SlotsPerEpoch))
		mix, err := RandaoMix(state, test.epoch)
		require.NoError(t, err)
		uniqueNumber := params.BeaconConfig().EpochsPerHistoricalVector + 1000
		binary.LittleEndian.PutUint64(mix, uniqueNumber)

		for _, mx := range randaoMixes {
			mxNum := bytesutil.FromBytes8(mx)
			assert.NotEqual(t, uniqueNumber, mxNum, "two distinct slices which have different representations in memory still contain the same value: %d", mxNum)
		}
	}
}

func TestGenerateSeed_OK(t *testing.T) {
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		intInBytes := make([]byte, 32)
		binary.LittleEndian.PutUint64(intInBytes, uint64(i))
		randaoMixes[i] = intInBytes
	}
	slot := 10 * params.BeaconConfig().MinSeedLookahead * params.BeaconConfig().SlotsPerEpoch
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		RandaoMixes: randaoMixes,
		Slot:        slot,
	})
	require.NoError(t, err)

	got, err := Seed(state, 10, params.BeaconConfig().DomainBeaconAttester)
	require.NoError(t, err)

	wanted := [32]byte{102, 82, 23, 40, 226, 79, 171, 11, 203, 23, 175, 7, 88, 202, 80,
		103, 68, 126, 195, 143, 190, 249, 210, 85, 138, 196, 158, 208, 11, 18, 136, 23}
	assert.Equal(t, wanted, got, "Incorrect generated seeds")
}
