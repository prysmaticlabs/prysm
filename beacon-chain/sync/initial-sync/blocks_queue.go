package initialsync

import (
	"context"
	"errors"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const (
	// queueStopCallTimeout is time allowed for queue to release resources when quitting.
	queueStopCallTimeout = 1 * time.Second
	// pollingInterval defines how often state machine needs to check for new events.
	pollingInterval = 200 * time.Millisecond
	// staleEpochTimeout is an period after which epoch's state is considered stale.
	staleEpochTimeout = 5 * pollingInterval
	// lookaheadEpochs is a default limit on how many forward epochs are loaded into queue.
	lookaheadEpochs = 4
)

var (
	errQueueCtxIsDone          = errors.New("queue's context is done, reinitialize")
	errQueueTakesTooLongToStop = errors.New("queue takes too long to stop")
	errNoEpochState            = errors.New("epoch state not found")
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

// blocksQueue is a priority queue that serves as a intermediary between block fetchers (producers)
// and block processing goroutine (consumer). Consumer can rely on order of incoming blocks.
type blocksQueue struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	highestExpectedSlot uint64
	state               *stateMachine
	blocksFetcher       blocksProvider
	headFetcher         blockchain.HeadFetcher
	fetchedBlocks       chan *eth.SignedBeaconBlock // output channel for ready blocks
	quit                chan struct{}               // termination notifier
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

	queue := &blocksQueue{
		ctx:                 ctx,
		cancel:              cancel,
		highestExpectedSlot: cfg.highestExpectedSlot,
		blocksFetcher:       blocksFetcher,
		headFetcher:         cfg.headFetcher,
		fetchedBlocks:       make(chan *eth.SignedBeaconBlock, allowedBlocksPerSecond),
		quit:                make(chan struct{}),
	}

	// Configure state machine.
	queue.state = newStateMachine()
	queue.state.addHandler(stateNew, eventSchedule, queue.onScheduleEvent(ctx))
	queue.state.addHandler(stateScheduled, eventDataReceived, queue.onDataReceivedEvent(ctx))
	queue.state.addHandler(stateDataParsed, eventReadyToSend, queue.onReadyToSendEvent(ctx))
	queue.state.addHandler(stateSkipped, eventExtendWindow, queue.onExtendWindowEvent(ctx))
	queue.state.addHandler(stateSent, eventCheckStale, queue.onCheckStaleEvent(ctx))

	return queue
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

	defer func() {
		q.blocksFetcher.stop()
		close(q.fetchedBlocks)
	}()

	if err := q.blocksFetcher.start(); err != nil {
		log.WithError(err).Debug("Can not start blocks provider")
	}

	startEpoch := helpers.SlotToEpoch(q.headFetcher.HeadSlot())
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch

	// Define epoch states as finite state machines.
	for i := startEpoch; i < startEpoch+lookaheadEpochs; i++ {
		q.state.addEpochState(i)
	}

	ticker := time.NewTicker(pollingInterval)
	tickerEvents := []eventID{eventSchedule, eventReadyToSend, eventCheckStale, eventExtendWindow}
	for {
		if q.headFetcher.HeadSlot() >= q.highestExpectedSlot {
			log.Debug("Highest expected slot reached")
			q.cancel()
		}

		select {
		case <-ticker.C:
			for _, state := range q.state.epochs {
				data := &fetchRequestParams{
					start: helpers.StartSlot(state.epoch),
					count: slotsPerEpoch,
				}

				// Trigger events on each epoch's state machine.
				for _, event := range tickerEvents {
					if err := q.state.trigger(event, state.epoch, data); err != nil {
						log.WithError(err).Debug("Can not trigger event")
					}
				}

				// Do garbage collection, and advance sliding window forward.
				if q.headFetcher.HeadSlot() >= helpers.StartSlot(state.epoch+1) {
					highestEpochSlot, err := q.state.highestEpochSlot()
					if err != nil {
						log.WithError(err).Debug("Cannot obtain highest epoch state number")
						continue
					}
					if err := q.state.removeEpochState(state.epoch); err != nil {
						log.WithError(err).Debug("Can not remove epoch state")
					}
					if len(q.state.epochs) < lookaheadEpochs {
						q.state.addEpochState(highestEpochSlot + 1)
					}
				}
			}
		case response, ok := <-q.blocksFetcher.requestResponses():
			if !ok {
				log.Debug("Fetcher closed output channel")
				q.cancel()
				return
			}
			// Update state of an epoch for which data is received.
			epoch := helpers.SlotToEpoch(response.start)
			ind := q.state.findEpochState(epoch)
			if ind < len(q.state.epochs) {
				state := q.state.epochs[ind]
				if err := q.state.trigger(eventDataReceived, state.epoch, response); err != nil {
					log.WithError(err).Debug("Can not trigger event")
					state.setState(stateNew)
					continue
				}
			}
		case <-q.ctx.Done():
			log.Debug("Context closed, exiting goroutine (blocks queue)")
			ticker.Stop()
			return
		}
	}
}

// onScheduleEvent is an event called on newly arrived epochs. Transforms state to scheduled.
func (q *blocksQueue) onScheduleEvent(ctx context.Context) eventHandlerFn {
	return func(es *epochState, in interface{}) (stateID, error) {
		data := in.(*fetchRequestParams)
		start := data.start
		count := mathutil.Min(data.count, q.highestExpectedSlot-start+1)
		if count <= 0 {
			return es.state, errStartSlotIsTooHigh
		}

		if err := q.blocksFetcher.scheduleRequest(ctx, start, count); err != nil {
			return es.state, err
		}
		return stateScheduled, nil
	}
}

// onDataReceivedEvent is an event called when data is received from fetcher.
func (q *blocksQueue) onDataReceivedEvent(ctx context.Context) eventHandlerFn {
	return func(es *epochState, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return es.state, ctx.Err()
		}

		response := in.(*fetchRequestResponse)
		if response.err != nil {
			return es.state, response.err
		}

		epoch := helpers.SlotToEpoch(response.start)
		ind := q.state.findEpochState(epoch)
		if ind >= len(q.state.epochs) {
			return es.state, errNoEpochState
		}
		q.state.epochs[ind].blocks = response.blocks
		return stateDataParsed, nil
	}
}

// onReadyToSendEvent is an event called to allow epochs with available blocks to send them downstream.
func (q *blocksQueue) onReadyToSendEvent(ctx context.Context) eventHandlerFn {
	return func(es *epochState, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return es.state, ctx.Err()
		}

		data := in.(*fetchRequestParams)
		epoch := helpers.SlotToEpoch(data.start)
		ind := q.state.findEpochState(epoch)
		if ind >= len(q.state.epochs) {
			return es.state, errNoEpochState
		}
		if len(q.state.epochs[ind].blocks) == 0 {
			return stateSkipped, nil
		}

		send := func() (stateID, error) {
			for _, block := range q.state.epochs[ind].blocks {
				select {
				case <-ctx.Done():
					return es.state, ctx.Err()
				case q.fetchedBlocks <- block:
				}
			}
			return stateSent, nil
		}

		// Make sure that we send epochs in a correct order.
		if q.state.isLowestEpochState(epoch) {
			return send()
		}

		// Make sure that previous epoch is already processed.
		for _, state := range q.state.epochs {
			// Review only previous slots.
			if state.epoch < epoch {
				switch state.state {
				case stateNew, stateScheduled, stateDataParsed:
					return es.state, nil
				default:
				}
			}
		}

		return send()
	}
}

// onExtendWindowEvent is and event allowing handlers to extend sliding window,
// in case where progress is not possible otherwise.
func (q *blocksQueue) onExtendWindowEvent(ctx context.Context) eventHandlerFn {
	return func(es *epochState, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return es.state, ctx.Err()
		}

		data := in.(*fetchRequestParams)
		epoch := helpers.SlotToEpoch(data.start)
		ind := q.state.findEpochState(epoch)
		if ind >= len(q.state.epochs) {
			return es.state, errNoEpochState
		}
		// Only the highest epoch with skipped state can trigger extension.
		highestEpochSlot, err := q.state.highestEpochSlot()
		if err != nil {
			return es.state, err
		}
		if highestEpochSlot != epoch {
			return es.state, nil
		}

		// Check if window is expanded recently, if so, time to reset and re-request the same blocks.
		resetWindow := false
		for _, state := range q.state.epochs {
			if state.state == stateSkippedExt {
				resetWindow = true
				break
			}
		}
		if resetWindow {
			for _, state := range q.state.epochs {
				state.setState(stateNew)
			}
			return stateNew, nil
		}

		// Extend sliding window.
		for i := 1; i <= lookaheadEpochs; i++ {
			q.state.addEpochState(highestEpochSlot + uint64(i))
		}
		return stateSkippedExt, nil
	}
}

// onCheckStaleEvent is an event that allows to mark stale epochs,
// so that they can be re-processed.
func (q *blocksQueue) onCheckStaleEvent(ctx context.Context) eventHandlerFn {
	return func(es *epochState, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return es.state, ctx.Err()
		}

		if time.Since(es.updated) > staleEpochTimeout {
			return stateSkipped, nil
		}

		return es.state, nil
	}
}
