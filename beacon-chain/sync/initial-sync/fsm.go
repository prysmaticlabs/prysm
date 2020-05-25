package initialsync

import (
	"errors"
	"fmt"
	"time"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

const (
	stateNew stateID = iota
	stateScheduled
	stateDataParsed
	stateSkipped
	stateSent
	stateSkippedExt
	stateComplete
)

const (
	eventSchedule eventID = iota
	eventDataReceived
	eventReadyToSend
	eventCheckStale
	eventExtendWindow
)

// stateID is unique handle for a state.
type stateID uint8

// eventID is unique handle for an event.
type eventID uint8

// stateMachine is a FSM that allows easy state transitions:
// State(S) x Event(E) -> Actions (A), State(S').
type stateMachine struct {
	epochs []*epochState
	events map[eventID]*stateMachineEvent
}

// epochState holds state of a single epoch.
type epochState struct {
	epoch   uint64
	state   stateID
	blocks  []*eth.SignedBeaconBlock
	updated time.Time
}

// stateMachineEvent is a container for event data.
type stateMachineEvent struct {
	name    eventID
	actions []*stateMachineAction
}

// stateMachineAction represents a state actions that can be attached to an event.
type stateMachineAction struct {
	state     stateID
	handlerFn eventHandlerFn
}

// eventHandlerFn is an event handler function's signature.
type eventHandlerFn func(*epochState, interface{}) (stateID, error)

// newStateMachine returns fully initialized state machine.
func newStateMachine() *stateMachine {
	return &stateMachine{
		epochs: make([]*epochState, 0, lookaheadEpochs),
		events: map[eventID]*stateMachineEvent{},
	}
}

// addHandler attaches an event handler to a state event.
func (sm *stateMachine) addHandler(state stateID, event eventID, fn eventHandlerFn) *stateMachineEvent {
	e, ok := sm.events[event]
	if !ok {
		e = &stateMachineEvent{
			name: event,
		}
		sm.events[event] = e
	}
	action := &stateMachineAction{
		state:     state,
		handlerFn: fn,
	}
	e.actions = append(e.actions, action)
	return e
}

// addEpochState allocates memory for tracking epoch state.
func (sm *stateMachine) addEpochState(epoch uint64) {
	state := &epochState{
		epoch:   epoch,
		state:   stateNew,
		blocks:  []*eth.SignedBeaconBlock{},
		updated: roughtime.Now(),
	}
	sm.epochs = append(sm.epochs, state)
}

// removeEpochState frees memory of processed epoch.
func (sm *stateMachine) removeEpochState(epoch uint64) error {
	ind, ok := sm.findEpochState(epoch)
	if !ok {
		return fmt.Errorf("state for %v epoch not found", epoch)
	}
	sm.epochs[ind].blocks = nil
	sm.epochs[ind] = sm.epochs[len(sm.epochs)-1]
	sm.epochs = sm.epochs[:len(sm.epochs)-1]
	return nil
}

// findEpochState returns index at which state.epoch = epoch, or len(epochs) if not found.
func (sm *stateMachine) findEpochState(epoch uint64) (int, bool) {
	for i, state := range sm.epochs {
		if epoch == state.epoch {
			return i, true
		}
	}
	return len(sm.epochs), false
}

// isLowestEpochState checks whether a given epoch is the lowest for which we know epoch state.
func (sm *stateMachine) isLowestEpochState(epoch uint64) bool {
	if _, ok := sm.findEpochState(epoch); !ok {
		return false
	}
	for _, state := range sm.epochs {
		if epoch > state.epoch {
			return false
		}
	}
	return true
}

// lowestEpoch returns epoch number for the earliest known epoch.
func (sm *stateMachine) lowestEpoch() (uint64, error) {
	if len(sm.epochs) == 0 {
		return 0, errors.New("no epoch states exist")
	}
	lowestEpoch := sm.epochs[0].epoch
	for _, state := range sm.epochs {
		if state.epoch < lowestEpoch {
			lowestEpoch = state.epoch
		}
	}
	return lowestEpoch, nil
}

// highestEpoch returns epoch number for the latest known epoch.
func (sm *stateMachine) highestEpoch() (uint64, error) {
	if len(sm.epochs) == 0 {
		return 0, errors.New("no epoch states exist")
	}
	highestEpoch := sm.epochs[0].epoch
	for _, state := range sm.epochs {
		if state.epoch > highestEpoch {
			highestEpoch = state.epoch
		}
	}
	return highestEpoch, nil
}

// String returns human readable representation of a state.
func (sm *stateMachine) String() string {
	return fmt.Sprintf("%v", sm.epochs)
}

// String returns human-readable representation of an epoch state.
func (es *epochState) String() string {
	return fmt.Sprintf("%d:%s", es.epoch, es.state)
}

// String returns human-readable representation of a state.
func (s stateID) String() (state string) {
	switch s {
	case stateNew:
		state = "new"
	case stateScheduled:
		state = "scheduled"
	case stateDataParsed:
		state = "dataParsed"
	case stateSkipped:
		state = "skipped"
	case stateSkippedExt:
		state = "skippedExt"
	case stateSent:
		state = "sent"
	case stateComplete:
		state = "complete"
	}
	return
}

// trigger invokes the event on a given epoch's state machine.
func (es *epochState) trigger(event *stateMachineEvent, data interface{}) error {
	if es == nil || event == nil {
		return errors.New("epoch state or event is nil")
	}
	for _, action := range event.actions {
		if action.state != es.state {
			continue
		}
		state, err := action.handlerFn(es, data)
		if err != nil {
			return err
		}
		es.setState(state)
	}
	return nil
}

// setState updates the current state of a given epoch.
func (es *epochState) setState(name stateID) {
	if es.state == name {
		return
	}
	es.updated = roughtime.Now()
	es.state = name
}

// String returns human-readable representation of an event.
func (e eventID) String() (event string) {
	switch e {
	case eventSchedule:
		event = "schedule"
	case eventDataReceived:
		event = "dataReceived"
	case eventReadyToSend:
		event = "readyToSend"
	case eventCheckStale:
		event = "checkStale"
	case eventExtendWindow:
		event = "extendWindow"
	}
	return
}
