package initialsync

import (
	"context"
	"fmt"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type blocksProviderMock struct {
}

func (f *blocksProviderMock) start() error {
	return nil
}

func (f *blocksProviderMock) stop() {
}

func (f *blocksProviderMock) scheduleRequest(ctx context.Context, start, count uint64) error {
	return nil
}

func (f *blocksProviderMock) requestResponses() <-chan *fetchRequestResponse {
	return nil
}

func TestBlocksQueueInitStartStop(t *testing.T) {
	mc, p2p, beaconDB := initializeTestServices(t, []uint64{}, []*peerData{})
	defer dbtest.TeardownDB(t, beaconDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
		headFetcher: mc,
		p2p:         p2p,
	})

	t.Run("stop without start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})

		if err := queue.stop(); err == nil {
			t.Errorf("expected error: %v", errQueueTakesTooLongToStop)
		}
	})

	t.Run("use default fetcher", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}
	})

	t.Run("stop timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}
		if err := queue.stop(); err == nil {
			t.Errorf("expected error: %v", errQueueTakesTooLongToStop)
		}
	})

	t.Run("check for leaked goroutines", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})

		if err := queue.start(); err != nil {
			t.Error(err)
		}
		// Blocks up until all resources are reclaimed (or timeout is called)
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
		select {
		case <-queue.fetchedBlocks:
		default:
			t.Error("queue.fetchedBlocks channel is leaked")
		}
		select {
		case <-fetcher.fetchResponses:
		default:
			t.Error("fetcher.fetchResponses channel is leaked")
		}
	})

	t.Run("re-starting of stopped queue", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
		if err := queue.start(); err == nil {
			t.Errorf("expected error not returned: %v", errQueueCtxIsDone)
		}
	})

	t.Run("multiple stopping attempts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}

		if err := queue.stop(); err != nil {
			t.Error(err)
		}
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})
		if err := queue.start(); err != nil {
			t.Error(err)
		}

		cancel()
		if err := queue.stop(); err != nil {
			t.Error(err)
		}
	})
}

func TestBlocksQueueUpdateSchedulerState(t *testing.T) {
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, 241),
		peers:              []*peerData{},
	}

	mc, _, beaconDB := initializeTestServices(t, chainConfig.expectedBlockSlots, chainConfig.peers)
	defer dbtest.TeardownDB(t, beaconDB)

	setupQueue := func(ctx context.Context) *blocksQueue {
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       &blocksProviderMock{},
			headFetcher:         mc,
			highestExpectedSlot: uint64(len(chainConfig.expectedBlockSlots)),
		})

		return queue
	}
	assertState := func(state *schedulerState, pending, valid, skipped, failed uint64) error {
		s := state.requestedBlocks
		res := s.pending != pending || s.valid != valid || s.skipped != skipped || s.failed != failed
		if res {
			b := struct{ pending, valid, skipped, failed uint64 }{pending, valid, skipped, failed,}
			return fmt.Errorf("invalid state, want: %+v, got: %+v", b, state.requestedBlocks)
		}
		return nil
	}

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		cancel()
		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if err := queue.scheduleFetchRequests(ctx); err != ctx.Err() {
			t.Errorf("expected error: %v", ctx.Err())
		}
	})

	t.Run("empty state on pristine node", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if state.currentSlot != 0 {
			t.Errorf("invalid current slot, want: %v, got: %v", 0, state.currentSlot)
		}
	})

	t.Run("empty state on pre-synced node", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		syncToSlot := uint64(7)
		setBlocksFromCache(ctx, t, mc, syncToSlot)
		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if state.currentSlot != syncToSlot {
			t.Errorf("invalid current slot, want: %v, got: %v", syncToSlot, state.currentSlot)
		}
	})

	t.Run("reset block batch size to default", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}

		// On enough valid blocks, batch size should get back to default value.
		state.blockBatchSize *= 2
		state.requestedBlocks.valid = blockBatchSize
		state.requestedBlocks.pending = 13
		state.requestedBlocks.skipped = 17
		state.requestedBlocks.failed = 19
		if err := assertState(state, 13, blockBatchSize, 17, 19); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}

		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, 13+state.blockBatchSize, 0, 17+blockBatchSize, 19); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}
	})

	t.Run("increase block batch size on too many failures", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}

		// On too many failures, batch size should get doubled and counters reset.
		state.requestedBlocks.valid = 19
		state.requestedBlocks.pending = 13
		state.requestedBlocks.skipped = 17
		state.requestedBlocks.failed = blockBatchSize
		if err := assertState(state, 13, 19, 17, blockBatchSize); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}

		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}
		if err := assertState(state, state.blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
	})

	t.Run("reset counters and block batch size on too many cached items", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}

		// On too many cached items, batch size and counters should reset.
		state.requestedBlocks.valid = queueMaxCachedBlocks
		state.requestedBlocks.pending = 13
		state.requestedBlocks.skipped = 17
		state.requestedBlocks.failed = 19
		if err := assertState(state, 13, queueMaxCachedBlocks, 17, 19); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}

		// This call should trigger resetting.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}
		if err := assertState(state, state.blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
	})

	t.Run("no pending blocks left", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}

		// On too many cached items, batch size and counters should reset.
		state.blockBatchSize = 2 * blockBatchSize
		state.requestedBlocks.pending = 0
		state.requestedBlocks.valid = 1
		state.requestedBlocks.skipped = 1
		state.requestedBlocks.failed = 1
		if err := assertState(state, 0, 1, 1, 1); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}

		// This call should trigger resetting.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpected batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}
		if err := assertState(state, state.blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
	})
}

func TestBlocksQueueScheduleFetchRequests(t *testing.T) {
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, 241),
		peers: []*peerData{
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
		},
	}

	mc, _, beaconDB := initializeTestServices(t, chainConfig.expectedBlockSlots, chainConfig.peers)
	defer dbtest.TeardownDB(t, beaconDB)

	setupQueue := func(ctx context.Context) *blocksQueue {
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       &blocksProviderMock{},
			headFetcher:         mc,
			highestExpectedSlot: uint64(len(chainConfig.expectedBlockSlots)),
		})

		return queue
	}
	assertState := func(state *schedulerState, pending, valid, skipped, failed uint64) error {
		s := state.requestedBlocks
		res := s.pending != pending || s.valid != valid || s.skipped != skipped || s.failed != failed
		if res {
			b := struct{ pending, valid, skipped, failed uint64 }{pending, valid, skipped, failed,}
			return fmt.Errorf("invalid state, want: %+v, got: %+v", b, state.requestedBlocks)
		}
		return nil
	}

	t.Run("check start/count boundaries", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		// Move sliding window normally.
		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}
		end := queue.highestExpectedSlot / state.blockBatchSize
		for i := uint64(0); i < end; i++ {
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
			if err := assertState(state, (i+1)*blockBatchSize, 0, 0, 0); err != nil {
				t.Error(err)
			}
		}

		// Make sure that the last request is up to highest expected slot.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, queue.highestExpectedSlot, 0, 0, 0); err != nil {
			t.Error(err)
		}

		// Try schedule beyond the highest slot.
		if err := queue.scheduleFetchRequests(ctx); err == nil {
			t.Errorf("expected error: %v", errStartSlotIsTooHigh)
		}
		if err := assertState(state, queue.highestExpectedSlot, 0, 0, 0); err != nil {
			t.Error(err)
		}
	})

	t.Run("too many failures", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		// Schedule enough items.
		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}
		end := queue.highestExpectedSlot / state.blockBatchSize
		for i := uint64(0); i < end; i++ {
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
			if err := assertState(state, (i+1)*blockBatchSize, 0, 0, 0); err != nil {
				t.Error(err)
			}
		}

		// "Process" some items and reschedule.
		if err := assertState(state, end*blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
		state.incrementCounter(failedBlockCounter, 25)
		if err := assertState(state, end*blockBatchSize-25, 0, 0, 25); err != nil {
			t.Error(err)
		}
		state.incrementCounter(failedBlockCounter, 500) // too high value shouldn't cause issues
		if err := assertState(state, 0, 0, 0, end*blockBatchSize); err != nil {
			t.Error(err)
		}

		// Due to failures, resetting is expected.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, 2*blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}
	})

	t.Run("too many skipped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		// Schedule enough items.
		if err := assertState(state, 0, 0, 0, 0); err != nil {
			t.Error(err)
		}
		end := queue.highestExpectedSlot / state.blockBatchSize
		for i := uint64(0); i < end; i++ {
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
			if err := assertState(state, (i+1)*blockBatchSize, 0, 0, 0); err != nil {
				t.Error(err)
			}
		}

		// "Process" some items and reschedule.
		if err := assertState(state, end*blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
		state.incrementCounter(skippedBlockCounter, 25)
		if err := assertState(state, end*blockBatchSize-25, 0, 25, 0); err != nil {
			t.Error(err)
		}
		state.incrementCounter(skippedBlockCounter, 500) // too high value shouldn't cause issues
		if err := assertState(state, 0, 0, end*blockBatchSize, 0); err != nil {
			t.Error(err)
		}

		// No pending items, resetting is expected (both counters and block batch size).
		state.blockBatchSize = 2 * blockBatchSize
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}
	})

	t.Run("reset block batch size", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		state.requestedBlocks.failed = blockBatchSize

		// Increase block batch size.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, 2*blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}

		// Reset block batch size.
		state.requestedBlocks.valid = blockBatchSize
		state.requestedBlocks.pending = 1
		state.requestedBlocks.failed = 1
		state.requestedBlocks.skipped = 1
		if err := assertState(state, 1, blockBatchSize, 1, 1); err != nil {
			t.Error(err)
		}
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, blockBatchSize+1, 0, blockBatchSize+1, 1); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}
	})

	t.Run("overcrowded scheduler", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		state.requestedBlocks.pending = queueMaxCachedBlocks
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		if err := assertState(state, blockBatchSize, 0, 0, 0); err != nil {
			t.Error(err)
		}
		if state.blockBatchSize != blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", blockBatchSize, state.blockBatchSize)
		}
	})
}

func TestBlocksQueueParseFetchResponse(t *testing.T) {
	chainConfig := struct {
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		expectedBlockSlots: makeSequence(1, 2*blockBatchSize*queueMaxPendingRequests+31),
		peers: []*peerData{
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
			{
				blocks:         makeSequence(1, 320),
				finalizedEpoch: 8,
				headSlot:       320,
			},
		},
	}

	mc, _, beaconDB := initializeTestServices(t, chainConfig.expectedBlockSlots, chainConfig.peers)
	defer dbtest.TeardownDB(t, beaconDB)

	setupQueue := func(ctx context.Context) *blocksQueue {
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       &blocksProviderMock{},
			headFetcher:         mc,
			highestExpectedSlot: uint64(len(chainConfig.expectedBlockSlots)),
		})

		return queue
	}

	var blocks []*eth.SignedBeaconBlock
	for i := 1; i <= blockBatchSize; i++ {
		blocks = append(blocks, &eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Slot: uint64(i),
			},
		})
	}

	t.Run("response error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)

		response := &fetchRequestResponse{
			start:  1,
			count:  blockBatchSize,
			blocks: blocks,
			err:    errStartSlotIsTooHigh,
		}
		if _, err := queue.parseFetchResponse(ctx, response); err != errStartSlotIsTooHigh {
			t.Errorf("expected error not thrown, want: %v, got: %v", errStartSlotIsTooHigh, err)
		}
	})

	t.Run("context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		queue := setupQueue(ctx)

		cancel()
		response := &fetchRequestResponse{
			start:  1,
			count:  blockBatchSize,
			blocks: blocks,
			err:    ctx.Err(),
		}
		if _, err := queue.parseFetchResponse(ctx, response); err != ctx.Err() {
			t.Errorf("expected error not thrown, want: %v, got: %v", ctx.Err(), err)
		}
	})

	t.Run("no skipped blocks", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)

		for i := uint64(1); i <= blockBatchSize; i++ {
			if _, ok := queue.state.cachedBlocks[i]; ok {
				t.Errorf("unexpeced block found: %v", i)
			}
		}

		response := &fetchRequestResponse{
			start:  1,
			count:  blockBatchSize,
			blocks: blocks,
		}
		if _, err := queue.parseFetchResponse(ctx, response); err != nil {
			t.Error(err)
		}

		// All blocks should be saved at this point.
		for i := uint64(1); i <= blockBatchSize; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock == nil {
				t.Errorf("unexpectedly marked as skipped: %v", i)
			}
		}
	})

	t.Run("with skipped blocks", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)

		for i := uint64(1); i <= blockBatchSize; i++ {
			if _, ok := queue.state.cachedBlocks[i]; ok {
				t.Errorf("unexpeced block found: %v", i)
			}
		}

		response := &fetchRequestResponse{
			start:  1,
			count:  blockBatchSize,
			blocks: blocks,
		}
		skipStart, skipEnd := uint64(5), uint64(15)
		response.blocks = append(response.blocks[:skipStart], response.blocks[skipEnd:]...)
		if _, err := queue.parseFetchResponse(ctx, response); err != nil {
			t.Error(err)
		}

		for i := skipStart + 1; i <= skipEnd; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock != nil {
				t.Errorf("unexpectedly marked as not skipped: %v", i)
			}
		}
		for i := uint64(1); i <= skipStart; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock == nil {
				t.Errorf("unexpectedly marked as skipped: %v", i)
			}
		}
		for i := skipEnd + 1; i <= blockBatchSize; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock == nil {
				t.Errorf("unexpectedly marked as skipped: %v", i)
			}
		}
	})
}

func TestBlocksQueueLoop(t *testing.T) {
	tests := []struct {
		name                string
		highestExpectedSlot uint64
		expectedBlockSlots  []uint64
		peers               []*peerData
	}{
		{
			name:                "Single peer with all blocks",
			highestExpectedSlot: 251,
			expectedBlockSlots:  makeSequence(1, 251),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
		},
		{
			name:                "Multiple peers with all blocks",
			highestExpectedSlot: 251,
			expectedBlockSlots:  makeSequence(1, 251),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
		},
		{
			name:                "Multiple peers with skipped slots",
			highestExpectedSlot: 576,
			expectedBlockSlots:  append(makeSequence(1, 64), makeSequence(500, 576)...), // up to 18th epoch
			peers: []*peerData{
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
			},
		},
		{
			name:                "Multiple peers with failures",
			highestExpectedSlot: 128,
			expectedBlockSlots:  makeSequence(1, 128),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
					failureSlots:   makeSequence(32*3+1, 32*3+32),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
					failureSlots:   makeSequence(1, 32*3),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc, p2p, beaconDB := initializeTestServices(t, tt.expectedBlockSlots, tt.peers)
			defer dbtest.TeardownDB(t, beaconDB)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fetcher := newBlocksFetcher(ctx, &blocksFetcherConfig{
				headFetcher: mc,
				p2p:         p2p,
			})
			queue := newBlocksQueue(ctx, &blocksQueueConfig{
				blocksFetcher:       fetcher,
				headFetcher:         mc,
				highestExpectedSlot: tt.highestExpectedSlot,
			})
			if err := queue.start(); err != nil {
				t.Error(err)
			}
			processBlock := func(block *eth.SignedBeaconBlock) error {
				if !beaconDB.HasBlock(ctx, bytesutil.ToBytes32(block.Block.ParentRoot)) {
					return fmt.Errorf("beacon node doesn't have a block in db with root %#x", block.Block.ParentRoot)
				}
				if featureconfig.Get().InitSyncNoVerify {
					if err := mc.ReceiveBlockNoVerify(ctx, block); err != nil {
						return err
					}
				} else {
					if err := mc.ReceiveBlockNoPubsubForkchoice(ctx, block); err != nil {
						return err
					}
				}

				return nil
			}

			var blocks []*eth.SignedBeaconBlock
			for block := range queue.fetchedBlocks {
				if err := processBlock(block); err != nil {
					queue.state.scheduler.incrementCounter(failedBlockCounter, 1)
					continue
				}
				blocks = append(blocks, block)
				queue.state.scheduler.incrementCounter(validBlockCounter, 1)
			}

			if err := queue.stop(); err != nil {
				t.Error(err)
			}

			if queue.headFetcher.HeadSlot() < uint64(len(tt.expectedBlockSlots)) {
				t.Errorf("Not enough slots synced, want: %v, got: %v",
					len(tt.expectedBlockSlots), queue.headFetcher.HeadSlot())
			}
			if len(blocks) != len(tt.expectedBlockSlots) {
				t.Errorf("Processes wrong number of blocks. Wanted %d got %d", len(tt.expectedBlockSlots), len(blocks))
			}
			var receivedBlockSlots []uint64
			for _, blk := range blocks {
				receivedBlockSlots = append(receivedBlockSlots, blk.Block.Slot)
			}
			if missing := sliceutil.NotUint64(sliceutil.IntersectionUint64(tt.expectedBlockSlots, receivedBlockSlots), tt.expectedBlockSlots); len(missing) > 0 {
				t.Errorf("Missing blocks at slots %v", missing)
			}
		})
	}
}

func setBlocksFromCache(ctx context.Context, t *testing.T, mc *mock.ChainService, highestSlot uint64) {
	cache.RLock()
	parentRoot := cache.rootCache[0]
	cache.RUnlock()
	for slot := uint64(0); slot <= highestSlot; slot++ {
		blk := &eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Slot:       slot,
				ParentRoot: parentRoot[:],
			},
		}
		mc.BlockNotifier().BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: blockfeed.ReceivedBlockData{
				SignedBlock: blk,
			},
		})

		if err := mc.ReceiveBlockNoPubsubForkchoice(ctx, blk); err != nil {
			t.Error(err)
		}

		currRoot, _ := ssz.HashTreeRoot(blk.Block)
		//logrus.Infof("block with slot %d , signing root %#x and parent root %#x", slot, currRoot, parentRoot)
		parentRoot = currRoot
	}
}
