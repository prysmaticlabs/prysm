package initialsync

import (
	"context"
	"fmt"
	"testing"

	"github.com/kevinms/leakybucket-go"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
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
			t.Logf("Isolated non-skipped slot found in %d iterations", i)
		}
	})
}
