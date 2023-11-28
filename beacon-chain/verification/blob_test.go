package verification

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestBlobIndexInBounds(t *testing.T) {
	ini := &Initializer{}
	_, blobs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 0, 1)
	b := blobs[0]
	// set Index to a value that is out of bounds
	v := ini.NewBlobVerifier(nil, b, GossipSidecarRequirements...)
	require.NoError(t, v.BlobIndexInBounds())

	b.Index = fieldparams.MaxBlobsPerBlock
	v = ini.NewBlobVerifier(nil, b, GossipSidecarRequirements...)
	require.ErrorIs(t, v.BlobIndexInBounds(), ErrBlobIndexInBounds)
}

func TestSlotBelowMaxDisparity(t *testing.T) {
	now := time.Now()
	// make genesis 1 slot in the past
	genesis := now.Add(-12 * time.Second)

	_, blobs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 0, 1)
	b := blobs[0]
	// slot 1 should be 12 seconds after genesis
	b.SignedBlockHeader.Header.Slot = 1

	// This clock will give a current slot of 1 on the nose
	happyClock := startup.NewClock(genesis, [32]byte{}, startup.WithNower(func() time.Time { return now }))
	ini := Initializer{shared: &sharedResources{clock: happyClock}}
	v := ini.NewBlobVerifier(nil, b, GossipSidecarRequirements...)
	require.NoError(t, v.SlotBelowMaxDisparity())

	// Since we have an early return for slots that are directly equal, give a time that is less than max disparity
	// but still in the previous slot.
	closeClock := startup.NewClock(genesis, [32]byte{}, startup.WithNower(func() time.Time { return now.Add(-1 * params.BeaconNetworkConfig().MaximumGossipClockDisparity / 2) }))
	ini = Initializer{shared: &sharedResources{clock: closeClock}}
	v = ini.NewBlobVerifier(nil, b, GossipSidecarRequirements...)
	require.NoError(t, v.SlotBelowMaxDisparity())

	// This clock will give a current slot of 0, with now coming more than max clock disparity before slot 1
	disparate := now.Add(-2 * params.BeaconNetworkConfig().MaximumGossipClockDisparity)
	dispClock := startup.NewClock(genesis, [32]byte{}, startup.WithNower(func() time.Time { return disparate }))
	// Set up initializer to use the clock that will set now to a little to far before slot 1
	ini = Initializer{shared: &sharedResources{clock: dispClock}}
	v = ini.NewBlobVerifier(nil, b, GossipSidecarRequirements...)
	require.ErrorIs(t, v.SlotBelowMaxDisparity(), ErrSlotBelowMaxDisparity)
}
