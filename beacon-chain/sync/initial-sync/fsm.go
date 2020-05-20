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

// stateID is unique handle for a state.
type stateID uint8

// eventID is unique handle for an event.
type eventID uint8

// stateMachineManager is a collection of managed FSMs.
type stateMachineManager struct {
	machines []*stateMachine
	events   map[eventID]*stateMachineEvent
}

// stateMachine holds a state of a single block range processing FSM.
// Each FSM allows deterministic state transitions:
// State(S) x Event(E) -> Actions (A), State(S').
type stateMachine struct {
	start   uint64
	count   uint64
	state   stateID
	blocks  []*eth.SignedBeaconBlock
	updated time.Time
}

// stateMachineEvent is a container for event data.
type stateMachineEvent struct {
	name    eventID
	actions map[stateID][]eventHandlerFn
}

// TODO remove me
//// stateMachineAction represents actions that can be attached to an event.
//// Action is applied when the FSM is in an expected start state only.
//type stateMachineAction struct {
//	state     stateID
//	handlerFn eventHandlerFn
//}

// eventHandlerFn is an event handler function's signature.
type eventHandlerFn func(*stateMachine, interface{}) (stateID, error)

// newStateMachineManager returns fully initialized state machine manager.
func newStateMachineManager() *stateMachineManager {
	return &stateMachineManager{
		machines: []*stateMachine{},
		events:   map[eventID]*stateMachineEvent{},
	}
}

// addHandler attaches an event handler to a state event.
func (sm *stateMachineManager) addEventHandler(event eventID, state stateID, fn eventHandlerFn) *stateMachineEvent {
	e, ok := sm.events[event]
	if !ok {
		e = &stateMachineEvent{
			name:    event,
			actions: make(map[stateID][]eventHandlerFn),
		}
		sm.events[event] = e
	}
	e.actions[state] = append(e.actions[state], fn)
	return e
}

// addStateMachine allocates memory for new FSM.
// Each machine is  tracking state of a given range of blocks.
func (sm *stateMachineManager) addStateMachine(start, count uint64) {
	fsm := &stateMachine{
		start:   start,
		count:   count,
		state:   stateNew,
		blocks:  []*eth.SignedBeaconBlock{},
		updated: roughtime.Now(),
	}
	sm.machines = append(sm.machines, fsm)
}

// removeStateMachine frees memory of a processed/finished FSM.
func (sm *stateMachineManager) removeStateMachine(fsm *stateMachine) error {
	if fsm == nil {
		return errors.New("nil machine")
	}
	ind, ok := sm.findStateMachineByStartBlock(fsm.start)
	if !ok {
		return fmt.Errorf("state for (%v, %v) machine is not found", fsm.start, fsm.count)
	}
	sm.machines[ind].blocks = nil
	sm.machines[ind] = sm.machines[len(sm.machines)-1]
	sm.machines = sm.machines[:len(sm.machines)-1]
	return nil
}

// findStateMachineByStartBlock returns index at which fsm.start = start,
// or len(sm.machines) if not found.
func (sm *stateMachineManager) findStateMachineByStartBlock(start uint64) (int, bool) {
	for i, fsm := range sm.machines {
		if start == fsm.start {
			return i, true
		}
	}
	return len(sm.machines), false
}

// isLowestStartBlock checks whether a given start block is the lowest for which we have a FSM.
func (sm *stateMachineManager) isLowestStartBlock(start uint64) bool {
	if _, ok := sm.findStateMachineByStartBlock(start); !ok {
		return false
	}
	for _, fsm := range sm.machines {
		if start > fsm.start {
			return false
		}
	}
	return true
}

// lowestStartBlock returns block number for the earliest known start block.
func (sm *stateMachineManager) lowestStartBlock() (uint64, error) {
	if len(sm.machines) == 0 {
		return 0, errors.New("no state machine exist")
	}
	lowestStartBlock := sm.machines[0].start
	for _, fsm := range sm.machines {
		if fsm.start < lowestStartBlock {
			lowestStartBlock = fsm.start
		}
	}
	return lowestStartBlock, nil
}

// highestStartBlock returns block number for the latest known start block.
func (sm *stateMachineManager) highestStartBlock() (uint64, error) {
	if len(sm.machines) == 0 {
		return 0, errors.New("no state machine exist")
	}
	highestStartBlock := sm.machines[0].start
	for _, fsm := range sm.machines {
		if fsm.start > highestStartBlock {
			highestStartBlock = fsm.start
		}
	}
	return highestStartBlock, nil
}

// String returns human readable representation of a FSM collection.
func (sm *stateMachineManager) String() string {
	return fmt.Sprintf("%v", sm.machines)
}

// setState updates the current state of a given FSM.
func (m *stateMachine) setState(name stateID) {
	if m.state == name {
		return
	}
	m.updated = roughtime.Now()
	m.state = name
}

// trigger invokes the event on a given state machine.
func (m *stateMachine) trigger(event *stateMachineEvent, data interface{}) error {
	if m == nil || event == nil {
		return errors.New("state machine or event is nil")
	}
	handlerFns, ok := event.actions[m.state]
	if !ok {
		return nil
	}
	for _, handlerFn := range handlerFns {
		state, err := handlerFn(m, data)
		if err != nil {
			return err
		}
		if m.state != state {
			m.setState(state)
			// No need to apply other actions if machines state has changed
			// (actions are not applicable to machine anymore)
			break
		}
	}
	return nil
}

// String returns human-readable representation of a FSM state.
func (m *stateMachine) String() string {
	return fmt.Sprintf("%d:%s", m.start, m.state)
}
