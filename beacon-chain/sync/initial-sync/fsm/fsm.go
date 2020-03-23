package fsm

import (
	"fmt"
	"time"
)

type StateMachine struct {
	currentState StateID
	updateTime   time.Time
	states       map[StateID]*State
	events       map[EventID]*Event
}

type HandlerFn func(*StateMachine, interface{}) (StateID, error)

type StateID string

type State struct {
	name StateID
	//handlerFns []HandlerFn
}

type EventID string

type Event struct {
	name    EventID
	actions []*Action
}

type Action struct {
	state     StateID
	handlerFn HandlerFn
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		states: map[StateID]*State{},
		events: map[EventID]*Event{},
	}
}

func (sm *StateMachine) String() string {
	return fmt.Sprintf("%v", sm.currentState)
}

func (sm *StateMachine) StateAge() time.Duration {
	return time.Since(sm.updateTime)
}

func (sm *StateMachine) SetCurrentState(name StateID) {
	if sm.currentState == name {
		return
	}
	sm.updateTime = time.Now()
	sm.currentState = name
	//if state, ok := sm.states[name]; ok {
	//	for _, fn := range state.handlerFns {
	//		if err := fn(sm); err != nil {
	//			return err
	//		}
	//	}
	//}
}

func (sm *StateMachine) CurrentState() StateID {
	return sm.currentState
}

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

//func (sm *StateMachine) AddHandler(name StateID, fn func(interface{}) error) *StateMachine {
//	state, ok := sm.states[name]
//	if !ok {
//		state = &State{
//			name: name,
//		}
//		sm.states[name] = state
//	}
//	sm.states[name].handlerFns = append(sm.states[name].handlerFns, fn)
//	return sm
//}

func (e *Event) AddHandler(state StateID, fn HandlerFn) *Event {
	action := &Action{
		state:     state,
		handlerFn: fn,
	}
	e.actions = append(e.actions, action)
	return e
}
