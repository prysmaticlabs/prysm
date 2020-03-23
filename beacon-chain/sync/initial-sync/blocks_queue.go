package initialsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/fsm"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

const (
	// queueMaxPendingRequests limits how many concurrent fetch request queue can initiate.
	queueMaxPendingRequests = 8
	// queueStopCallTimeout is time allowed for queue to release resources when quitting.
	queueStopCallTimeout = 1 * time.Second

	fsmPollingInterval = 200 * time.Millisecond
	lookaheadEpochs    = 4
	staleEpochTimeout  = 2 * fsmPollingInterval
)

const (
	stateNew          = "new"
	stateScheduled    = "scheduled"
	stateDataReceived = "dataReceived"
	stateDataParsed   = "dataParsed"
	stateSkipped      = "skipped"
	stateSkippedExt   = "skippedExt"
	stateSent         = "sent"
	statePartial      = "partial"
	stateComplete     = "complete"
	stateGarbage      = "garbage"
)
const (
	eventSchedule       = "schedule"
	eventDataReceived   = "dataReceived"
	eventReadyToSend    = "readyToSend"
	eventCheckProcessed = "checkProcessed"
	eventMoveForward    = "moveForward"
	eventExtendWindow   = "extendWindow"
	eventUnmarkSkipped  = "unmarkSkipped"
)

var (
	errQueueCtxIsDone          = errors.New("queue's context is done, reinitialize")
	errQueueTakesTooLongToStop = errors.New("queue takes too long to stop")
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
	epochStates         map[uint64]*fsm.StateMachine
	epochBlocks         map[uint64][]*eth.SignedBeaconBlock
	epochStatesKeys     []uint64
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

	return &blocksQueue{
		ctx:                 ctx,
		cancel:              cancel,
		highestExpectedSlot: cfg.highestExpectedSlot,
		epochStates:         make(map[uint64]*fsm.StateMachine, lookaheadEpochs),
		epochStatesKeys:     []uint64{},
		epochBlocks:         make(map[uint64][]*eth.SignedBeaconBlock, lookaheadEpochs),
		blocksFetcher:       blocksFetcher,
		headFetcher:         cfg.headFetcher,
		fetchedBlocks:       make(chan *eth.SignedBeaconBlock, allowedBlocksPerSecond),
		quit:                make(chan struct{}),
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

	defer func() {
		q.blocksFetcher.stop()
		close(q.fetchedBlocks)
	}()

	if err := q.blocksFetcher.start(); err != nil {
		log.WithError(err).Debug("Can not start blocks provider")
	}

	startEpoch := helpers.SlotToEpoch(q.headFetcher.HeadSlot())
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch

	// Define epochStates as finite state machines.
	for i := startEpoch; i < startEpoch+lookaheadEpochs; i++ {
		q.addEpochStateMachine(q.ctx, i)
	}

	ticker := time.NewTicker(fsmPollingInterval)
	tickerEvents := []fsm.EventID{
		eventSchedule, eventReadyToSend, eventCheckProcessed, eventMoveForward, eventExtendWindow}

	for {
		if q.headFetcher.HeadSlot() >= q.highestExpectedSlot {
			log.Debug("Highest expected slot reached")
			q.cancel()
		}

		log.WithFields(logrus.Fields{
			"epochStates": q.epochStates,
		}).Debug("Tick-tack")
		select {
		case <-ticker.C:
			for _, epoch := range q.epochStatesKeys {
				state := q.epochStates[epoch]
				data := &fetchRequestParams{
					start: helpers.StartSlot(epoch),
					count: slotsPerEpoch,
				}

				// Trigger regular events for each epoch's state machine.
				for _, event := range tickerEvents {
					if err := state.Trigger(event, data); err != nil {
						log.WithError(err).Debug("Can not trigger event")
					}
				}

				// Do garbage collection, and advance sliding window forward.
				if state.CurrentState() == stateGarbage {
					highestEpochState, err := q.highestEpochState()
					if err != nil {
						log.WithError(err).Debug("Cannot obtain highest epoch state number")
						continue
					}
					if err := q.removeEpochStateMachine(q.ctx, epoch); err != nil {
						log.WithError(err).Debug("Can not remove epoch state")
					}
					if len(q.epochStates) < lookaheadEpochs {
						q.addEpochStateMachine(q.ctx, highestEpochState+1)
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
			if state, ok := q.epochStates[epoch]; ok {
				if err := state.Trigger(eventDataReceived, response); err != nil {
					log.WithError(err).Debug("Can not trigger event")
					state.SetCurrentState(stateNew)
					continue
				}
				state.SetCurrentState(stateDataParsed)
			}
		case <-q.ctx.Done():
			log.Debug("Context closed, exiting goroutine (blocks queue)")
			ticker.Stop()
			return
		}
	}
}

func (q *blocksQueue) addEpochStateMachine(ctx context.Context, epoch uint64) *fsm.StateMachine {
	state := fsm.NewStateMachine()
	state.OnEvent(eventSchedule).AddHandler(stateNew, q.scheduleEventHandler(ctx))
	state.OnEvent(eventDataReceived).AddHandler(stateScheduled, q.dataReceivedEventHandler(ctx))
	state.OnEvent(
		eventReadyToSend,
	).AddHandler(stateDataParsed, q.readyToSendEventHandler(ctx),
	).AddHandler(stateDataParsed, q.readyToSendEventHandler(ctx))
	state.OnEvent(
		eventCheckProcessed,
	).AddHandler(stateSent, q.checkProcessedEventHandler(ctx),
	).AddHandler(stateSkipped, q.checkProcessedEventHandler(ctx),
	).AddHandler(stateSkippedExt, q.checkProcessedEventHandler(ctx))
	state.OnEvent(eventMoveForward).AddHandler(stateComplete, q.moveForwardEventHandler(ctx))
	state.OnEvent(eventExtendWindow).AddHandler(stateSkipped, q.extendWindowEventHandler(ctx))

	state.SetCurrentState(stateNew)

	q.epochStates[epoch] = state
	q.updateEpochStatesKeys()
	return state
}

func (q *blocksQueue) scheduleEventHandler(ctx context.Context) fsm.HandlerFn {
	return func(sm *fsm.StateMachine, in interface{}) (fsm.StateID, error) {
		data := in.(*fetchRequestParams)
		start := data.start
		count := mathutil.Min(data.count, q.highestExpectedSlot-start+1)
		if count <= 0 {
			return "", errStartSlotIsTooHigh
		}

		if err := q.blocksFetcher.scheduleRequest(ctx, start, count); err != nil {
			return "", err
		}
		return stateScheduled, nil
	}
}

func (q *blocksQueue) dataReceivedEventHandler(ctx context.Context) fsm.HandlerFn {
	return func(sm *fsm.StateMachine, in interface{}) (fsm.StateID, error) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		response := in.(*fetchRequestResponse)
		if response.err != nil {
			return "", response.err
		}

		epoch := helpers.SlotToEpoch(response.start)
		q.epochBlocks[epoch] = response.blocks
		log.WithFields(logrus.Fields{
			"epoch": epoch,
			"start": response.start,
		}).Debug("Data cached")

		return stateDataReceived, nil
	}
}

func (q *blocksQueue) readyToSendEventHandler(ctx context.Context) fsm.HandlerFn {
	return func(sm *fsm.StateMachine, in interface{}) (fsm.StateID, error) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		data := in.(*fetchRequestParams)
		epoch := helpers.SlotToEpoch(data.start)
		cachedBlocks, ok := q.epochBlocks[epoch]
		if !ok {
			return "", errors.New("no cache found")
		}

		if len(cachedBlocks) == 0 {
			return stateSkipped, nil
		}

		// Make sure that we send epochs in correct order.
		readyToSend := false
		lowestEpochState, err := q.lowestEpochState()
		if err != nil {
			return sm.CurrentState(), err
		}
		if epoch == lowestEpochState {
			readyToSend = true
		} else if sm, ok := q.epochStates[epoch-1]; ok {
			switch sm.CurrentState() {
			case stateNew, stateScheduled, stateDataReceived, stateDataParsed:
			default:
				readyToSend = true
			}
		}
		if !readyToSend {
			return sm.CurrentState(), nil
		}

		for _, block := range cachedBlocks {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case q.fetchedBlocks <- block:
			}
		}

		return stateSent, nil
	}
}

func (q *blocksQueue) checkProcessedEventHandler(ctx context.Context) fsm.HandlerFn {
	return func(sm *fsm.StateMachine, in interface{}) (fsm.StateID, error) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		data := in.(*fetchRequestParams)
		if q.headFetcher.HeadSlot() >= data.start+data.count {
			return stateComplete, nil
		}

		if sm.StateAge() > staleEpochTimeout {
			return stateSkipped, nil
		}

		return sm.CurrentState(), nil
	}
}

func (q *blocksQueue) moveForwardEventHandler(ctx context.Context) fsm.HandlerFn {
	return func(sm *fsm.StateMachine, in interface{}) (fsm.StateID, error) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		data := in.(*fetchRequestParams)
		epoch := helpers.SlotToEpoch(data.start)
		if _, ok := q.epochStates[epoch]; !ok {
			return sm.CurrentState(), fmt.Errorf("epoch %v has no cached state", epoch)
		}

		return stateGarbage, nil
	}
}

func (q *blocksQueue) extendWindowEventHandler(ctx context.Context) fsm.HandlerFn {
	return func(sm *fsm.StateMachine, in interface{}) (fsm.StateID, error) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		data := in.(*fetchRequestParams)
		epoch := helpers.SlotToEpoch(data.start)
		if _, ok := q.epochStates[epoch]; !ok {
			return sm.CurrentState(), fmt.Errorf("epoch %v has no cached state", epoch)
		}
		highestEpochState, err := q.highestEpochState()
		if err != nil {
			return sm.CurrentState(), err
		}
		if highestEpochState != epoch {
			return sm.CurrentState(), nil
		}
		// Make sure that all the previous states are stuck as well.
		resetWindow := false
		for _, epoch := range q.epochStates {
			curState := epoch.CurrentState()
			if curState != stateSkipped && curState != stateSkippedExt {
				return sm.CurrentState(), nil
			}
			if curState == stateSkippedExt {
				resetWindow = true
			}
		}
		// We have already expanded, time to reset, and re-request the same blocks.
		if resetWindow {
			for _, state := range q.epochStates {
				state.SetCurrentState(stateNew)
			}
			return stateNew, nil
		}
		// Extend sliding window, immediately request within extended epoch.
		q.addEpochStateMachine(ctx, highestEpochState+1)
		q.addEpochStateMachine(ctx, highestEpochState+2)

		return stateSkippedExt, nil
	}
}

func (q *blocksQueue) updateEpochStatesKeys() {
	keys := make([]uint64, 0)
	for key := range q.epochStates {
		keys = append(keys, key)
	}
	q.epochStatesKeys = keys
}

func (q *blocksQueue) removeEpochStateMachine(ctx context.Context, epoch uint64) error {
	if _, ok := q.epochStates[epoch]; !ok {
		return fmt.Errorf("epoch %v has no cached state", epoch)
	}
	delete(q.epochStates, epoch)
	q.updateEpochStatesKeys()
	return nil
}

func (q *blocksQueue) highestEpochState() (uint64, error) {
	if len(q.epochStatesKeys) == 0 {
		return 0, errors.New("no epoch states exist")
	}
	highestEpoch := q.epochStatesKeys[0]
	for epoch := range q.epochStates {
		if epoch > highestEpoch {
			highestEpoch = epoch
		}
	}
	return highestEpoch, nil
}

func (q *blocksQueue) lowestEpochState() (uint64, error) {
	if len(q.epochStatesKeys) == 0 {
		return 0, errors.New("no epoch states exist")
	}
	lowestEpoch := q.epochStatesKeys[0]
	for epoch := range q.epochStates {
		if epoch < lowestEpoch {
			lowestEpoch = epoch
		}
	}
	return lowestEpoch, nil
}
