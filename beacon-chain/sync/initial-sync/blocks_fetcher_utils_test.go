package initialsync

import (
	"context"
	"fmt"
	"testing"

	"github.com/kevinms/leakybucket-go"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBlocksFetcher_nonSkippedSlotAfter(t *testing.T) {
	peersGen := func(size int) []*peerData {
		blocks := append(makeSequence(1, 64), makeSequence(500, 640)...)
		blocks = append(blocks, makeSequence(51200, 51264)...)
		blocks = append(blocks, 55000)
		blocks = append(blocks, makeSequence(57000, 57256)...)
		var peersData []*peerData
		for i := 0; i < size; i++ {
			peersData = append(peersData, &peerData{
				blocks:         blocks,
				finalizedEpoch: 1800,
				headSlot:       57000,
			})
		}
		return peersData
	}
	chainConfig := struct {
		peers []*peerData
	}{
		peers: peersGen(5),
	}

	mc, p2p, _ := initializeTestServices(t, []uint64{}, chainConfig.peers)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := newBlocksFetcher(
		ctx,
		&blocksFetcherConfig{
			finalizationFetcher: mc,
			headFetcher:         mc,
			p2p:                 p2p,
		},
	)
	fetcher.rateLimiter = leakybucket.NewCollector(6400, 6400, false)
	seekSlots := map[uint64]uint64{
		0:     1,
		10:    11,
		31:    32,
		32:    33,
		63:    64,
		64:    500,
		160:   500,
		352:   500,
		480:   500,
		512:   513,
		639:   640,
		640:   51200,
		6640:  51200,
		51200: 51201,
	}
	for seekSlot, expectedSlot := range seekSlots {
		t.Run(fmt.Sprintf("range: %d (%d-%d)", expectedSlot-seekSlot, seekSlot, expectedSlot), func(t *testing.T) {
			slot, err := fetcher.nonSkippedSlotAfter(ctx, seekSlot)
			assert.NoError(t, err)
			assert.Equal(t, expectedSlot, slot, "Unexpected slot")
		})
	}

	t.Run("test isolated non-skipped slot", func(t *testing.T) {
		seekSlot := uint64(51264)
		expectedSlot := uint64(55000)
		found := false
		var i int
		for i = 0; i < 100; i++ {
			slot, err := fetcher.nonSkippedSlotAfter(ctx, seekSlot)
			assert.NoError(t, err)
			if slot == expectedSlot {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Isolated non-skipped slot not found in %d iterations: %v", i, expectedSlot)
		} else {
			log.Debugf("Isolated non-skipped slot found in %d iterations", i)
		}
	})

	t.Run("no peers with higher target epoch available", func(t *testing.T) {
		peers := []*peerData{
			{finalizedEpoch: 3, headSlot: 160},
			{finalizedEpoch: 3, headSlot: 160},
			{finalizedEpoch: 3, headSlot: 160},
			{finalizedEpoch: 8, headSlot: 320},
			{finalizedEpoch: 8, headSlot: 320},
			{finalizedEpoch: 10, headSlot: 320},
			{finalizedEpoch: 10, headSlot: 640},
		}
		p2p := p2pt.NewTestP2P(t)
		connectPeers(t, p2p, peers, p2p.Peers())
		fetcher := newBlocksFetcher(
			ctx,
			&blocksFetcherConfig{
				headFetcher:         mc,
				finalizationFetcher: mc,
				p2p:                 p2p,
			},
		)
		mc.FinalizedCheckPoint = &eth.Checkpoint{
			Epoch: 10,
		}
		require.NoError(t, mc.State.SetSlot(12*params.BeaconConfig().SlotsPerEpoch))

		fetcher.mode = modeStopOnFinalizedEpoch
		slot, err := fetcher.nonSkippedSlotAfter(ctx, 160)
		assert.ErrorContains(t, errSlotIsTooHigh.Error(), err)
		assert.Equal(t, uint64(0), slot)

		fetcher.mode = modeNonConstrained
		require.NoError(t, mc.State.SetSlot(20*params.BeaconConfig().SlotsPerEpoch))
		slot, err = fetcher.nonSkippedSlotAfter(ctx, 160)
		assert.ErrorContains(t, errSlotIsTooHigh.Error(), err)
		assert.Equal(t, uint64(0), slot)
	})
}

func TestBlocksFetcher_currentHeadAndTargetEpochs(t *testing.T) {
	tests := []struct {
		name               string
		syncMode           syncMode
		peers              []*peerData
		ourFinalizedEpoch  uint64
		ourHeadSlot        uint64
		expectedHeadEpoch  uint64
		targetEpoch        uint64
		targetEpochSupport int
	}{
		{
			name:               "ignore lower epoch peers in best finalized",
			syncMode:           modeStopOnFinalizedEpoch,
			ourHeadSlot:        5 * params.BeaconConfig().SlotsPerEpoch,
			expectedHeadEpoch:  4,
			ourFinalizedEpoch:  4,
			targetEpoch:        10,
			targetEpochSupport: 3,
			peers: []*peerData{
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
			},
		},
		{
			name:               "resolve ties in best finalized",
			syncMode:           modeStopOnFinalizedEpoch,
			ourHeadSlot:        5 * params.BeaconConfig().SlotsPerEpoch,
			expectedHeadEpoch:  4,
			ourFinalizedEpoch:  4,
			targetEpoch:        10,
			targetEpochSupport: 3,
			peers: []*peerData{
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
			},
		},
		{
			name:               "best non-finalized",
			syncMode:           modeNonConstrained,
			ourHeadSlot:        5 * params.BeaconConfig().SlotsPerEpoch,
			expectedHeadEpoch:  5,
			ourFinalizedEpoch:  4,
			targetEpoch:        20,
			targetEpochSupport: 1,
			peers: []*peerData{
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 3, headSlot: 160},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 8, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 10, headSlot: 320},
				{finalizedEpoch: 15, headSlot: 640},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, p2p, _ := initializeTestServices(t, []uint64{}, tt.peers)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fetcher := newBlocksFetcher(
				ctx,
				&blocksFetcherConfig{
					headFetcher:         mc,
					finalizationFetcher: mc,
					p2p:                 p2p,
				},
			)
			mc.FinalizedCheckPoint = &eth.Checkpoint{
				Epoch: tt.ourFinalizedEpoch,
			}
			require.NoError(t, mc.State.SetSlot(tt.ourHeadSlot))
			fetcher.mode = tt.syncMode

			// Head and target epochs calculation.
			headEpoch, targetEpoch, peers := fetcher.calculateHeadAndTargetEpochs()
			assert.Equal(t, tt.expectedHeadEpoch, headEpoch, "Unexpected head epoch")
			assert.Equal(t, tt.targetEpoch, targetEpoch, "Unexpected target epoch")
			assert.Equal(t, tt.targetEpochSupport, len(peers), "Unexpected number of peers supporting target epoch")

			// Best finalized and non-finalized slots.
			finalizedSlot := tt.targetEpoch * params.BeaconConfig().SlotsPerEpoch
			if tt.syncMode == modeStopOnFinalizedEpoch {
				assert.Equal(t, finalizedSlot, fetcher.bestFinalizedSlot(), "Unexpected finalized slot")
			} else {
				assert.Equal(t, finalizedSlot, fetcher.bestNonFinalizedSlot(), "Unexpected non-finalized slot")
			}
		})
	}
}
