package initialsync

import (
	"context"
	"fmt"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
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

	t.Run("check for leaked goroutines", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		queue := newBlocksQueue(ctx, &blocksQueueConfig{
			blocksFetcher:       fetcher,
			headFetcher:         mc,
			highestExpectedSlot: blockBatchSize,
		})
		err := queue.start()
		if err != nil {
			t.Error(err)
		}
		queue.stop() // should block up until all resources are reclaimed
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
		queue.stop()
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

		queue.stop()
		queue.stop()
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
		queue.stop()
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

	hook := logTest.NewGlobal()
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

		// Move sliding window normally.
		for i := 0; i < 7; i++ {
			hook.Reset()
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
			testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
			testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", i*blockBatchSize+1))
			testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", uint64(i+1)*blockBatchSize))
		}

		// Make sure that the last request is up to highest expected slot.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", 7*blockBatchSize+1))
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", queue.highestExpectedSlot))

		// Try schedule beyond the highest slot.
		hook.Reset()
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Queue's start position is too high, resetting counters")
		testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", 1))
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", blockBatchSize))
	})

	t.Run("too many failures", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		// Schedule enough items.
		for i := 0; i < 8; i++ {
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
		}

		// "Process" some items and reschedule.
		if err := assertState(state, 241, 0, 0, 0); err != nil {
			t.Error(err)
		}
		state.updateFailedBlocksCounter(50)
		if err := assertState(state, 191, 0, 0, 50); err != nil {
			t.Error(err)
		}
		state.updateFailedBlocksCounter(500) // too high value shouldn't cause issues
		if err := assertState(state, 0, 0, 0, 241); err != nil {
			t.Error(err)
		}

		// Due to failures, resetting is expected. But there are pending items.
		hook.Reset()
		state.updateFailedBlocksCounter(1)
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Too many failures, increasing block batch size")
		if err := assertState(state, 64, 0, 0, 0); err != nil { // make sure that reset occurs
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}

		// Now, everything should be scheduled ok.
		for i := 1; i < 3; i++ {
			hook.Reset()
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
			testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
			testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", i*blockBatchSize*2+1))
			testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", uint64(i+1)*blockBatchSize*2))
		}

		// Make sure that the last request is up to highest expected slot.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", 3*blockBatchSize*2+1))
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", queue.highestExpectedSlot))

	})

	t.Run("too many skipped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		// Schedule enough items.
		for i := 0; i < 8; i++ {
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
		}

		// "Process" some items and reschedule.
		if err := assertState(state, 241, 0, 0, 0); err != nil {
			t.Error(err)
		}
		state.updateSkippedBlocksCounter(50)
		if err := assertState(state, 191, 0, 50, 0); err != nil {
			t.Error(err)
		}
		state.updateSkippedBlocksCounter(500) // too high value shouldn't cause issues
		if err := assertState(state, 0, 0, 241, 0); err != nil {
			t.Error(err)
		}

		// No problems for sliding window, as pending items are updated.
		for i := 0; i < 3; i++ {
			hook.Reset()
			if err := queue.scheduleFetchRequests(ctx); err != nil {
				t.Error(err)
			}
			testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
			testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", i*2*blockBatchSize+1))
			testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", uint64(i+1)*blockBatchSize*2))
		}

		// Make sure that the last request is up to highest expected slot.
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Fetch request scheduled")
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("start=%d", 3*2*blockBatchSize+1))
		testutil.AssertLogsContain(t, hook, fmt.Sprintf("pending:%d", queue.highestExpectedSlot))

	})

	t.Run("reset block batch size", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		queue := setupQueue(ctx)
		state := queue.state.scheduler

		state.requestedBlocks.failed = blockBatchSize

		// Increase block batch size.
		hook.Reset()
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Too many failures, increasing block batch size")
		if err := assertState(state, 64, 0, 0, 0); err != nil { // make sure that reset occurs
			t.Error(err)
		}
		if state.blockBatchSize != 2*blockBatchSize {
			t.Errorf("unexpeced block batch size, want: %v, got: %v", 2*blockBatchSize, state.blockBatchSize)
		}

		// Reset block batch size.
		state.requestedBlocks.valid = 2 * blockBatchSize
		state.requestedBlocks.pending = 0
		if err := assertState(state, 0, 2*blockBatchSize, 0, 0); err != nil { // make sure that reset occurs
			t.Error(err)
		}
		hook.Reset()
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Many valid blocks, time to reset block batch size")
		if err := assertState(state, 32, 0, 0, 0); err != nil { // make sure that reset occurs
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

		state.requestedBlocks.pending = queueMaxPendingBlocks

		hook.Reset()
		if err := queue.scheduleFetchRequests(ctx); err != nil {
			t.Error(err)
		}
		testutil.AssertLogsContain(t, hook, "Overcrowded scheduler counters, resetting")
		if err := assertState(state, 32, 0, 0, 0); err != nil { // make sure that reset occurs
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
		if err := queue.parseFetchResponse(ctx, response); err != errStartSlotIsTooHigh {
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
		if err := queue.parseFetchResponse(ctx, response); err != ctx.Err() {
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
		if err := queue.parseFetchResponse(ctx, response); err != nil {
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
		if err := queue.parseFetchResponse(ctx, response); err != nil {
			t.Error(err)
		}

		for i := skipStart + 1; i <= skipEnd; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock!= nil {
				t.Errorf("unexpectedly marked as not skipped: %v", i)
			}
		}
		for i := uint64(1); i <= skipStart; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock== nil {
				t.Errorf("unexpectedly marked as skipped: %v", i)
			}
		}
		for i := skipEnd + 1; i <= blockBatchSize; i++ {
			block, ok := queue.state.cachedBlocks[i]
			if !ok {
				t.Errorf("expeced block not found: %v", i)
			}
			if block.SignedBeaconBlock== nil {
				t.Errorf("unexpectedly marked as skipped: %v", i)
			}
		}
	})
}
