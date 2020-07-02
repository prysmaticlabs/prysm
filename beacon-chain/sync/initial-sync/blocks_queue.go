package initialsync

import (
	"context"
	"errors"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/sirupsen/logrus"
)

const (
	// queueStopCallTimeout is time allowed for queue to release resources when quitting.
	queueStopCallTimeout = 1 * time.Second
	// pollingInterval defines how often state machine needs to check for new events.
	pollingInterval = 200 * time.Millisecond
	// staleEpochTimeout is an period after which epoch's state is considered stale.
	staleEpochTimeout = 1 * time.Second
	// lookaheadSteps is a limit on how many forward steps are loaded into queue.
	// Each step is managed by assigned finite state machine.
	lookaheadSteps = 8
)

var (
	errQueueCtxIsDone             = errors.New("queue's context is done, reinitialize")
	errQueueTakesTooLongToStop    = errors.New("queue takes too long to stop")
	errInvalidInitialState        = errors.New("invalid initial state")
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
	smm                 *stateMachineManager
	blocksFetcher       *blocksFetcher
	headFetcher         blockchain.HeadFetcher
	highestExpectedSlot uint64
	fetchedBlocks       chan []*eth.SignedBeaconBlock // output channel for ready blocks
	quit                chan struct{}                 // termination notifier
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
		fetchedBlocks:       make(chan []*eth.SignedBeaconBlock),
		quit:                make(chan struct{}),
	}

	// Configure state machines.
	queue.smm = newStateMachineManager()
	queue.smm.addEventHandler(eventTick, stateNew, queue.onScheduleEvent(ctx))
	queue.smm.addEventHandler(eventDataReceived, stateScheduled, queue.onDataReceivedEvent(ctx))
	queue.smm.addEventHandler(eventTick, stateDataParsed, queue.onReadyToSendEvent(ctx))
	queue.smm.addEventHandler(eventTick, stateSkipped, queue.onProcessSkippedEvent(ctx))
	queue.smm.addEventHandler(eventTick, stateSent, queue.onCheckStaleEvent(ctx))

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

	// Define initial state machines.
	startSlot := q.headFetcher.HeadSlot()
	blocksPerRequest := q.blocksFetcher.blocksPerSecond
	for i := startSlot; i < startSlot+blocksPerRequest*lookaheadSteps; i += blocksPerRequest {
		q.smm.addStateMachine(i)
	}

	ticker := time.NewTicker(pollingInterval)
	for {
		// Check highest expected slot when we approach chain's head slot.
		if q.headFetcher.HeadSlot() >= q.highestExpectedSlot {
			// By the time initial sync is complete, highest slot may increase, re-check.
			if q.highestExpectedSlot < q.blocksFetcher.bestFinalizedSlot() {
				q.highestExpectedSlot = q.blocksFetcher.bestFinalizedSlot()
				continue
			}
			log.WithField("slot", q.highestExpectedSlot).Debug("Highest expected slot reached")
			q.cancel()
		}

		select {
		case <-ticker.C:
			for _, key := range q.smm.keys {
				fsm := q.smm.machines[key]
				if err := fsm.trigger(eventTick, nil); err != nil {
					log.WithFields(logrus.Fields{
						"event": eventTick,
						"epoch": helpers.SlotToEpoch(fsm.start),
						"start": fsm.start,
						"error": err.Error(),
					}).Debug("Can not trigger event")
				}
				// Do garbage collection, and advance sliding window forward.
				if q.headFetcher.HeadSlot() >= fsm.start+blocksPerRequest-1 {
					highestStartBlock, err := q.smm.highestStartBlock()
					if err != nil {
						log.WithError(err).Debug("Cannot obtain highest epoch state number")
						continue
					}
					if err := q.smm.removeStateMachine(fsm.start); err != nil {
						log.WithError(err).Debug("Can not remove state machine")
					}
					if len(q.smm.machines) < lookaheadSteps {
						q.smm.addStateMachine(highestStartBlock + blocksPerRequest)
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
			if fsm, ok := q.smm.findStateMachine(response.start); ok {
				if err := fsm.trigger(eventDataReceived, response); err != nil {
					log.WithFields(logrus.Fields{
						"event": eventDataReceived,
						"epoch": helpers.SlotToEpoch(fsm.start),
						"error": err.Error(),
					}).Debug("Can not process event")
					fsm.setState(stateNew)
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
	return func(m *stateMachine, in interface{}) (stateID, error) {
		if m.state != stateNew {
			return m.state, errInvalidInitialState
		}
		if m.start > q.highestExpectedSlot {
			m.setState(stateSkipped)
			return m.state, errSlotIsTooHigh
		}
		blocksPerRequest := q.blocksFetcher.blocksPerSecond
		if err := q.blocksFetcher.scheduleRequest(ctx, m.start, blocksPerRequest); err != nil {
			return m.state, err
		}
		return stateScheduled, nil
	}
}

// onDataReceivedEvent is an event called when data is received from fetcher.
func (q *blocksQueue) onDataReceivedEvent(ctx context.Context) eventHandlerFn {
	return func(m *stateMachine, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return m.state, ctx.Err()
		}
		if m.state != stateScheduled {
			return m.state, errInvalidInitialState
		}
		response, ok := in.(*fetchRequestResponse)
		if !ok {
			return 0, errInputNotFetchRequestParams
		}
		if response.err != nil {
			// Current window is already too big, re-request previous epochs.
			if response.err == errSlotIsTooHigh {
				for _, fsm := range q.smm.machines {
					if fsm.start < response.start && fsm.state == stateSkipped {
						fsm.setState(stateNew)
					}
				}
			}
			return m.state, response.err
		}
		m.blocks = response.blocks
		return stateDataParsed, nil
	}
}

// onReadyToSendEvent is an event called to allow epochs with available blocks to send them downstream.
func (q *blocksQueue) onReadyToSendEvent(ctx context.Context) eventHandlerFn {
	return func(m *stateMachine, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return m.state, ctx.Err()
		}
		if m.state != stateDataParsed {
			return m.state, errInvalidInitialState
		}

		if len(m.blocks) == 0 {
			return stateSkipped, nil
		}

		send := func() (stateID, error) {
			select {
			case <-ctx.Done():
				return m.state, ctx.Err()
			case q.fetchedBlocks <- m.blocks:
			}
			return stateSent, nil
		}

		// Make sure that we send epochs in a correct order.
		// If machine is the first (has lowest start block), send.
		if m.isFirst() {
			return send()
		}

		// Make sure that previous epoch is already processed.
		for _, fsm := range q.smm.machines {
			// Review only previous slots.
			if fsm.start < m.start {
				switch fsm.state {
				case stateNew, stateScheduled, stateDataParsed:
					return m.state, nil
				}
			}
		}

		return send()
	}
}

// onProcessSkippedEvent is an event triggered on skipped machines, allowing handlers to
// extend lookahead window, in case where progress is not possible otherwise.
func (q *blocksQueue) onProcessSkippedEvent(ctx context.Context) eventHandlerFn {
	return func(m *stateMachine, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return m.state, ctx.Err()
		}
		if m.state != stateSkipped {
			return m.state, errInvalidInitialState
		}

		// Only the highest epoch with skipped state can trigger extension.
		if !m.isLast() {
			// When a state machine stays in skipped state for too long - reset it.
			if time.Since(m.updated) > 5*staleEpochTimeout {
				return stateNew, nil
			}
			return m.state, nil
		}

		// Make sure that all machines are in skipped state i.e. manager cannot progress without reset or
		// moving the last machine's start block forward (in an attempt to find next non-skipped block).
		if !q.smm.allMachinesInState(stateSkipped) {
			return m.state, nil
		}

		// Shift start position of all the machines except for the last one.
		startSlot := q.headFetcher.HeadSlot() + 1
		blocksPerRequest := q.blocksFetcher.blocksPerSecond
		if err := q.smm.removeAllStateMachines(); err != nil {
			return stateSkipped, err
		}
		for i := startSlot; i < startSlot+blocksPerRequest*(lookaheadSteps-1); i += blocksPerRequest {
			q.smm.addStateMachine(i)
		}

		// Replace the last (currently activated) state machine.
		nonSkippedSlot, err := q.blocksFetcher.nonSkippedSlotAfter(
			ctx, startSlot+blocksPerRequest*(lookaheadSteps-1)-1)
		if err != nil {
			return stateSkipped, err
		}
		if q.highestExpectedSlot < q.blocksFetcher.bestFinalizedSlot() {
			q.highestExpectedSlot = q.blocksFetcher.bestFinalizedSlot()
		}
		if nonSkippedSlot > q.highestExpectedSlot {
			nonSkippedSlot = startSlot + blocksPerRequest*(lookaheadSteps-1)
		}
		q.smm.addStateMachine(nonSkippedSlot)
		return stateSkipped, nil
	}
}

// onCheckStaleEvent is an event that allows to mark stale epochs,
// so that they can be re-processed.
func (q *blocksQueue) onCheckStaleEvent(ctx context.Context) eventHandlerFn {
	return func(m *stateMachine, in interface{}) (stateID, error) {
		if ctx.Err() != nil {
			return m.state, ctx.Err()
		}
		if m.state != stateSent {
			return m.state, errInvalidInitialState
		}

		// Break out immediately if bucket is not stale.
		if time.Since(m.updated) < staleEpochTimeout {
			return m.state, nil
		}

		return stateSkipped, nil
	}
}
