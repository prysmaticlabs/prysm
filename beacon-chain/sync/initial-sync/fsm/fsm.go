package fsm

import (
	"fmt"
	"time"
)

// StateMachine is a FSM that allows easy state transitions:
// State(S) x Event(E) -> Actions (A), State(S').
type StateMachine struct {
	currentState StateID
	updateTime   time.Time
	states       map[StateID]*State
	events       map[EventID]*Event
}

// HandlerFn is an event handler signature.
type HandlerFn func(*StateMachine, interface{}) (StateID, error)

// StateID is unique handle for a state.
type StateID string

// EventID is unique handle for an event.
type EventID string

// State is a container for data.
type State struct {
	name StateID
}

// Event is a container for event data.
type Event struct {
	name    EventID
	actions []*Action
}

// Action represents event actions that can be attached to an event.
type Action struct {
	state     StateID
	handlerFn HandlerFn
}

// NewStateMachine returns fully initialized state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		states: map[StateID]*State{},
		events: map[EventID]*Event{},
	}
}

// String returns human readable representation of a state.
func (sm *StateMachine) String() string {
	return fmt.Sprintf("%v", sm.currentState)
}

// StateAge returns time since the current state has been updated.
func (sm *StateMachine) StateAge() time.Duration {
	return time.Since(sm.updateTime)
}

// SetCurrentState updates the current state.
func (sm *StateMachine) SetCurrentState(name StateID) {
	if sm.currentState == name {
		return
	}
	sm.updateTime = time.Now()
	sm.currentState = name
}

// CurrentState returns the current state.
func (sm *StateMachine) CurrentState() StateID {
	return sm.currentState
}

// Trigger is used to call events on a state machine.
func (sm *StateMachine) Trigger(name EventID, data interface{}) error {
	event, ok := sm.events[name]
	if !ok {
		return fmt.Errorf("event %v not found", name)
	}

	var actions []*Action
	for _, action := range event.actions {
		if sm.currentState == action.state {
			actions = append(actions, action)
		}
	}

	for _, action := range actions {
		state, err := action.handlerFn(sm, data)
		if err != nil {
			return err
		}
		sm.SetCurrentState(state)
	}

	return nil
}

// OnEvent is a helper method to attach event to the state machine.
func (sm *StateMachine) OnEvent(name EventID) *Event {
	event, ok := sm.events[name]
	if !ok {
		event = &Event{
			name: name,
		}
		sm.events[name] = event
	}
	return event
}

// AddHandler is a helper method to attach event handlers to events.
func (e *Event) AddHandler(state StateID, fn HandlerFn) *Event {
	action := &Action{
		state:     state,
		handlerFn: fn,
	}
	e.actions = append(e.actions, action)
	return e
}
