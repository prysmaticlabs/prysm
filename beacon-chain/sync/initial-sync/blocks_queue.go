package initialsync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/sirupsen/logrus"
)

const (
	queueMaxPendingRequests  = 4
	queueFetchRequestTimeout = 60 * time.Second
	queueMaxCachedBlocks     = 8 * queueMaxPendingRequests * blockBatchSize
	queueRefreshRate         = time.Second / 20
	queueStopCallTimeout     = 1 * time.Second
)

const (
	validBlockCounter = iota
	skippedBlockCounter
	failedBlockCounter
)

var (
	errQueueCtxIsDone          = errors.New("queue's context is done, reinitialize")
	errQueueTakesTooLongToStop = errors.New("queue takes to long to stop")
)

// blocksProvider exposes enough methods for queue to fetch incoming blocks.
type blocksProvider interface {
	requestResponses() <-chan *fetchRequestResponse
	scheduleRequest(ctx context.Context, start, count uint64) error
	start() error
	stop()
}

// blocksQueueConfig is a config to setup block queue service.
type blocksQueueConfig struct {
	blocksFetcher       blocksProvider
	headFetcher         blockchain.HeadFetcher
	startSlot           uint64
	highestExpectedSlot uint64
	p2p                 p2p.P2P
}

// blocksQueueState holds internal queue state (for easier management of state transitions).
type blocksQueueState struct {
	scheduler    *schedulerState
	sender       *senderState
	cachedBlocks map[uint64]*cachedBlock
}

// schedulerState a state of scheduling process.
type schedulerState struct {
	sync.Mutex
	currentSlot     uint64
	blockBatchSize  uint64
	requestedBlocks struct {
		pending, valid, skipped, failed uint64
	}
}

// senderState is a state of block sending process.
type senderState struct {
	sync.Mutex
}

// cachedBlock is a container for signed beacon block.
type cachedBlock struct {
	*eth.SignedBeaconBlock
}

// blocksQueue is a priority queue that serves as a intermediary between block fetchers (producers)
// and block processing goroutine (consumer). Consumer can rely on order of incoming blocks.
type blocksQueue struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	highestExpectedSlot  uint64
	state                *blocksQueueState
	blocksFetcher        blocksProvider
	headFetcher          blockchain.HeadFetcher
	fetchedBlocks        chan *eth.SignedBeaconBlock // output channel for ready blocks
	pendingFetchRequests chan struct{}               // pending requests semaphore
	pendingFetchedBlocks chan struct{}               // notifier, pings block sending handler
	quit                 chan struct{}               // termination notifier
}

// newBlocksQueue creates initialized priority queue.
func newBlocksQueue(ctx context.Context, cfg *blocksQueueConfig) *blocksQueue {
	ctx, cancel := context.WithCancel(ctx)

	blocksFetcher := cfg.blocksFetcher
	if blocksFetcher == nil {
		blocksFetcher = newBlocksFetcher(ctx, &blocksFetcherConfig{
			headFetcher: cfg.headFetcher,
			p2p:         cfg.p2p,
		})
	}

	return &blocksQueue{
		ctx:                 ctx,
		cancel:              cancel,
		highestExpectedSlot: cfg.highestExpectedSlot,
		state: &blocksQueueState{
			scheduler: &schedulerState{
				currentSlot:    cfg.startSlot,
				blockBatchSize: blockBatchSize,
			},
			sender:       &senderState{},
			cachedBlocks: make(map[uint64]*cachedBlock, queueMaxCachedBlocks),
		},
		blocksFetcher:        blocksFetcher,
		headFetcher:          cfg.headFetcher,
		fetchedBlocks:        make(chan *eth.SignedBeaconBlock, blockBatchSize),
		pendingFetchRequests: make(chan struct{}, queueMaxPendingRequests),
		pendingFetchedBlocks: make(chan struct{}, queueMaxPendingRequests),
		quit:                 make(chan struct{}),
	}
}

// start boots up the queue processing.
func (q *blocksQueue) start() error {
	select {
	case <-q.ctx.Done():
		return errQueueCtxIsDone
	default:
		go q.loop()
		return nil
	}
}

// stop terminates all queue operations.
func (q *blocksQueue) stop() error {
	q.cancel()
	select {
	case <-q.quit:
		return nil
	case <-time.After(queueStopCallTimeout):
		return errQueueTakesTooLongToStop
	}
}

// loop is a main queue loop.
func (q *blocksQueue) loop() {
	defer close(q.quit)

	// Wait for all goroutines to wrap up (forced by cancelled context), and do a cleanup.
	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
		q.blocksFetcher.stop()
		close(q.fetchedBlocks)
	}()

	if err := q.blocksFetcher.start(); err != nil {
		log.WithError(err).Debug("Can not start blocks provider")
	}

	for {
		if q.headFetcher.HeadSlot() >= q.highestExpectedSlot {
			log.Debug("Highest expected slot reached")
			return
		}

		select {
		case <-q.ctx.Done():
			log.Debug("Context closed, exiting goroutine (blocks queue)")
			return
		case q.pendingFetchRequests <- struct{}{}:
			wg.Add(1)
			go func() {
				defer func() {
					<-q.pendingFetchRequests             // notify semaphore
					q.pendingFetchedBlocks <- struct{}{} // notify sender of data availability
				}()
				defer wg.Done()

				// Schedule request.
				if err := q.scheduleFetchRequests(q.ctx); err != nil {
					log.WithError(err).Debug("Error scheduling fetch request")
					return
				}

				// Obtain response (if request is scheduled ok).
				select {
				case <-q.ctx.Done():
				case resp, ok := <-q.blocksFetcher.requestResponses():
					if !ok {
						log.Debug("Blocks fetcher closed output channel")
						q.cancel()
						return
					}
					// Process incoming response into blocks.
					if err := q.parseFetchResponse(q.ctx, resp); err != nil {
						q.state.scheduler.updateCounter(failedBlockCounter, resp.count)
						log.WithError(err).Debug("Error processing received blocks")
						return
					}
				}
			}()
		case <-q.pendingFetchedBlocks:
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := q.sendFetchedBlocks(q.ctx); err != nil {
					log.WithError(err).Debug("Error sending received blocks")
				}
			}()
		}
	}
}

// scheduleFetchRequests enqueues block fetch requests to block fetcher.
func (q *blocksQueue) scheduleFetchRequests(ctx context.Context) error {
	log.Debug("scheduleFetchRequests")

	// Depends on various factors (error rate, current head slot, number of skipped blocks).
	// State updates may include a temporary increase of block's batch size (to look ahead farther).
	if err := q.updateSchedulerState(ctx); err != nil {
		return err
	}

	q.state.scheduler.Lock()
	defer q.state.scheduler.Unlock()

	blocks := &q.state.scheduler.requestedBlocks
	offset := blocks.pending + blocks.skipped + blocks.failed + blocks.valid

	start := q.state.scheduler.currentSlot + offset + 1
	count := mathutil.Min(q.state.scheduler.blockBatchSize, q.highestExpectedSlot-start+1)
	if count <= 0 {
		log.WithFields(logrus.Fields{
			"state":               fmt.Sprintf("%+v", blocks),
			"start":               start,
			"count":               count,
			"highestExpectedSlot": q.highestExpectedSlot,
		}).Debug("Queue's start position is too high")
		return errStartSlotIsTooHigh
	}

	ctx, _ = context.WithTimeout(ctx, queueFetchRequestTimeout)
	if err := q.blocksFetcher.scheduleRequest(ctx, start, count); err != nil {
		return err
	}

	q.state.scheduler.requestedBlocks.pending += count
	log.WithFields(logrus.Fields{
		"state":               fmt.Sprintf("%+v", blocks),
		"start":               start,
		"count":               count,
		"highestExpectedSlot": q.highestExpectedSlot,
	}).Debug("Fetch request scheduled")

	return nil
}

// updateSchedulerState adjusts scheduler's state, so that it never stalls.
// Makes necessary adjustments to forward lookup window size and slot offset.
func (q *blocksQueue) updateSchedulerState(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	q.state.scheduler.Lock()
	defer q.state.scheduler.Unlock()
	s := q.state.scheduler

	log.WithFields(logrus.Fields{
		"state":          fmt.Sprintf("%+v", s.requestedBlocks),
		"blockBatchSize": s.blockBatchSize,
	}).Debug("Checking scheduler state")

	blocks := &s.requestedBlocks
	resetStateCounters := func() {
		q.state.sender.Lock()
		q.state.cachedBlocks = make(map[uint64]*cachedBlock, queueMaxCachedBlocks)
		q.state.sender.Unlock()

		blocks.pending, blocks.valid, blocks.failed, blocks.skipped = 0, 0, 0, 0
		s.currentSlot = q.headFetcher.HeadSlot()
	}

	// Update state's current slot pointer.
	if count := blocks.pending + blocks.skipped + blocks.failed + blocks.valid; count == 0 {
		s.currentSlot = q.headFetcher.HeadSlot()
		return nil
	}

	// Too many failures (blocks that can't be processed at this time).
	if blocks.failed >= s.blockBatchSize/2 {
		s.blockBatchSize *= 2
		resetStateCounters()

		log.WithFields(logrus.Fields{
			"counters":       fmt.Sprintf("%+v", blocks),
			"currentSlot":    s.currentSlot,
			"blockBatchSize": s.blockBatchSize,
		}).Debug("Too many unprocessable blocks, increasing block batch size")
		return nil
	}

	// Given enough valid blocks, we can set back block batch size.
	if blocks.valid >= blockBatchSize && s.blockBatchSize != blockBatchSize {
		blocks.skipped, blocks.valid = blocks.skipped+blocks.valid, 0
		s.blockBatchSize = blockBatchSize

		log.WithFields(logrus.Fields{
			"counters":       fmt.Sprintf("%+v", blocks),
			"currentSlot":    s.currentSlot,
			"blockBatchSize": s.blockBatchSize,
		}).Debug("Enough valid blocks, block batch size reset")
	}

	// Too many items in scheduler, time to update current slot to point to current head's slot.
	count := blocks.pending + blocks.skipped + blocks.failed + blocks.valid
	if count >= queueMaxCachedBlocks {
		s.blockBatchSize = blockBatchSize
		resetStateCounters()

		log.WithFields(logrus.Fields{
			"counters":       fmt.Sprintf("%+v", blocks),
			"currentSlot":    s.currentSlot,
			"blockBatchSize": s.blockBatchSize,
		}).Debug("Too many cached blocks, resetting")
		return nil
	}

	// All blocks processed, no pending blocks.
	if count := blocks.skipped + blocks.failed + blocks.valid; blocks.pending == 0 && count > 0 {
		s.blockBatchSize = blockBatchSize
		resetStateCounters()
		log.WithFields(logrus.Fields{
			"counters":       fmt.Sprintf("%+v", blocks),
			"currentSlot":    s.currentSlot,
			"blockBatchSize": s.blockBatchSize,
		}).Debug("No pending blocks, resetting counters")
		return nil
	}

	return nil
}

// parseFetchResponse processes incoming responses.
func (q *blocksQueue) parseFetchResponse(ctx context.Context, response *fetchRequestResponse) error {
	log.Debug("parseFetchResponse")
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if response.err != nil {
		return response.err
	}

	// Extract beacon blocks.
	responseBlocks := make(map[uint64]*eth.SignedBeaconBlock, len(response.blocks))
	for _, blk := range response.blocks {
		responseBlocks[blk.Block.Slot] = blk
	}

	q.state.sender.Lock()
	defer q.state.sender.Unlock()

	// Cache blocks in [start, start + count) range, include skipped blocks.
	var skippedBlocks uint64
	end := response.start + mathutil.Max(response.count, uint64(len(response.blocks)))
	for slot := response.start; slot < end; slot++ {
		if block, ok := responseBlocks[slot]; ok {
			q.state.cachedBlocks[slot] = &cachedBlock{
				SignedBeaconBlock: block,
			}
			delete(responseBlocks, slot)
			skippedBlocks++
			continue
		}
		q.state.cachedBlocks[slot] = &cachedBlock{}
	}

	// If there are any items left in incoming response, log them too.
	for slot, block := range responseBlocks {
		q.state.cachedBlocks[slot] = &cachedBlock{
			SignedBeaconBlock: block,
		}
	}

	// Update scheduler's counters.
	q.state.scheduler.updateCounter(skippedBlockCounter, skippedBlocks)

	return nil
}

// sendFetchedBlocks analyses available blocks, and sends them downstream in a correct slot order.
// Blocks are checked starting from the current head slot, and up until next consecutive block is available.
func (q *blocksQueue) sendFetchedBlocks(ctx context.Context) error {
	log.Debug("sendFetchedBlocks")
	q.state.sender.Lock()
	defer q.state.sender.Unlock()

	startSlot := q.headFetcher.HeadSlot() + 1
	nonSkippedSlot := uint64(0)
	for slot := startSlot; slot <= q.highestExpectedSlot; slot++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		blockData, ok := q.state.cachedBlocks[slot]
		if !ok {
			break
		}
		if blockData.SignedBeaconBlock != nil && blockData.Block != nil {
			q.fetchedBlocks <- blockData.SignedBeaconBlock
			nonSkippedSlot = slot
		}
	}

	// Remove processed blocks.
	if nonSkippedSlot > 0 {
		for key := startSlot; key <= nonSkippedSlot; key++ {
			delete(q.state.cachedBlocks, key)
		}
	}

	return nil
}

func (s *schedulerState) handleCurrentSlotUpdate(slot uint64) {
	s.Lock()
	defer s.Unlock()

	blocks := s.requestedBlocks
	if count := blocks.pending + blocks.skipped + blocks.failed + blocks.valid; count == 0 {
		s.currentSlot = slot
	}
}

// updateCounter updates scheduler's blocks counters.
func (s *schedulerState) updateCounter(counter int, n uint64) {
	s.Lock()
	defer s.Unlock()

	n = mathutil.Min(s.requestedBlocks.pending, n)
	s.requestedBlocks.pending -= n

	switch counter {
	case validBlockCounter:
		s.requestedBlocks.valid += n
	case skippedBlockCounter:
		s.requestedBlocks.skipped += n
	case failedBlockCounter:
		s.requestedBlocks.failed += n
	}
}
