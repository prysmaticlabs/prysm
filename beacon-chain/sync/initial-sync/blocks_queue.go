package initialsync

import (
	"context"
	"errors"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
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
	errQueueCtxIsDone             = errors.New("queue's context is done, reinitialize")
	errQueueTakesTooLongToStop    = errors.New("queue takes too long to stop")
	errNoEpochState               = errors.New("epoch state not found")
	errInputNotFetchRequestParams = errors.New("input data is not type *fetchRequestParams")
)

// blocksQueueConfig is a config to setup block queue service.
type blocksQueueConfig struct {
	blocksFetcher       *blocksFetcher
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
	blocksFetcher       *blocksFetcher
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
	highestExpectedSlot := cfg.highestExpectedSlot
	if highestExpectedSlot <= cfg.startSlot {
		highestExpectedSlot = blocksFetcher.bestFinalizedSlot()
	}

	queue := &blocksQueue{
		ctx:                 ctx,
		cancel:              cancel,
		highestExpectedSlot: highestExpectedSlot,
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
			// By the time initial sync is complete, highest slot may increase, re-check.
			if q.highestExpectedSlot < q.blocksFetcher.bestFinalizedSlot() {
				q.highestExpectedSlot = q.blocksFetcher.bestFinalizedSlot()
				continue
			}
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
						log.WithFields(logrus.Fields{
							"event": event,
							"epoch": state.epoch,
							"error": err.Error(),
						}).Debug("Can not trigger event")
					}
				}

				// Do garbage collection, and advance sliding window forward.
				if q.headFetcher.HeadSlot() >= helpers.StartSlot(state.epoch+1) {
					highestEpoch, err := q.state.highestEpoch()
					if err != nil {
						log.WithError(err).Debug("Cannot obtain highest epoch state number")
						continue
					}
					if err := q.state.removeEpochState(state.epoch); err != nil {
						log.WithError(err).Debug("Can not remove epoch state")
					}
					if len(q.state.epochs) < lookaheadEpochs {
						q.state.addEpochState(highestEpoch + 1)
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
			if ind, ok := q.state.findEpochState(epoch); ok {
				state := q.state.epochs[ind]
				if err := q.state.trigger(eventDataReceived, state.epoch, response); err != nil {
					log.WithFields(logrus.Fields{
						"event": eventDataReceived,
						"epoch": state.epoch,
						"error": err.Error(),
					}).Debug("Can not trigger event")
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
		data, ok := in.(*fetchRequestParams)
		if !ok {
			return 0, errInputNotFetchRequestParams
		}
		if err := q.blocksFetcher.scheduleRequest(ctx, data.start, data.count); err != nil {
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

		response, ok := in.(*fetchRequestResponse)
		if !ok {
			return 0, errInputNotFetchRequestParams
		}
		epoch := helpers.SlotToEpoch(response.start)
		if response.err != nil {
			// Current window is already too big, re-request previous epochs.
			if response.err == errSlotIsTooHigh {
				for _, state := range q.state.epochs {
					isSkipped := state.state == stateSkipped || state.state == stateSkippedExt
					if state.epoch < epoch && isSkipped {
						state.setState(stateNew)
					}
				}
			}
			return es.state, response.err
		}

		ind, ok := q.state.findEpochState(epoch)
		if !ok {
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

		data, ok := in.(*fetchRequestParams)
		if !ok {
			return 0, errInputNotFetchRequestParams
		}
		epoch := helpers.SlotToEpoch(data.start)
		ind, ok := q.state.findEpochState(epoch)
		if !ok {
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

		data, ok := in.(*fetchRequestParams)
		if !ok {
			return 0, errInputNotFetchRequestParams
		}
		epoch := helpers.SlotToEpoch(data.start)
		if _, ok := q.state.findEpochState(epoch); !ok {
			return es.state, errNoEpochState
		}

		// Only the highest epoch with skipped state can trigger extension.
		highestEpoch, err := q.state.highestEpoch()
		if err != nil {
			return es.state, err
		}
		if highestEpoch != epoch {
			return es.state, nil
		}

		// Check if window is expanded recently, if so, time to reset and re-request the same blocks.
		resetWindow := false
		skippedEpochs := 0
		for _, state := range q.state.epochs {
			if state.state == stateSkippedExt {
				resetWindow = true
				break
			}
			if state.state == stateSkipped || state.state == stateSkippedExt {
				skippedEpochs++
			}
		}
		// Reset if everything is skipped or extension took place during previous iteration.
		if resetWindow || (skippedEpochs == len(q.state.epochs)) {
			for _, state := range q.state.epochs {
				state.setState(stateNew)
			}
			// Fill the gaps between epochs.
			start, err := q.state.lowestEpoch()
			if err != nil {
				return es.state, err
			}
			end, err := q.state.highestEpoch()
			if err != nil {
				return es.state, err
			}
			for i := start; i < end; i++ {
				if _, ok := q.state.findEpochState(i); !ok {
					q.state.addEpochState(i)
				}
			}
			return stateNew, nil
		}

		// Extend sliding window.
		nonSkippedSlot, err := q.blocksFetcher.nonSkippedSlotAfter(ctx, helpers.StartSlot(highestEpoch+1))
		if err != nil {
			return es.state, err
		}
		if nonSkippedSlot > q.highestExpectedSlot {
			return es.state, nil
		}
		q.state.addEpochState(helpers.SlotToEpoch(nonSkippedSlot))
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
