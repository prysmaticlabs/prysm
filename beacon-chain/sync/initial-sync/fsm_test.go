package initialsync

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestStateMachineManager_String(t *testing.T) {
	tests := []struct {
		name     string
		machines map[uint64]*stateMachine
		want     string
	}{
		{
			"empty epoch state list",
			map[uint64]*stateMachine{},
			"map[]",
		},
		{
			"newly created state machine",
			map[uint64]*stateMachine{
				0:   {start: 64 * 0, state: stateNew},
				64:  {start: 64 * 1, state: stateScheduled},
				128: {start: 64 * 2, state: stateDataParsed},
				196: {start: 64 * 3, state: stateSkipped},
				256: {start: 64 * 4, state: stateSent},
			},
			"map[0:{0:new} 64:{2:scheduled} 128:{4:dataParsed} 196:{6:skipped} 256:{8:sent}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smm := &stateMachineManager{
				machines: tt.machines,
			}
			if got := smm.String(); got != tt.want {
				t.Errorf("unexpected output,  got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestStateMachine_StateIDString(t *testing.T) {
	stateIDs := []stateID{stateNew, stateScheduled, stateDataParsed, stateSkipped, stateSent}
	want := "[new scheduled dataParsed skipped sent]"
	if got := fmt.Sprintf("%v", stateIDs); got != want {
		t.Errorf("unexpected output, got: %q, want: %q", got, want)
	}
}

func TestStateMachine_EventIDString(t *testing.T) {
	eventIDs := []eventID{eventTick, eventDataReceived}
	want := "[tick dataReceived]"
	if got := fmt.Sprintf("%v", eventIDs); got != want {
		t.Errorf("unexpected output, got: %q, want: %q", got, want)
	}
}

func TestStateMachineManager_addEventHandler(t *testing.T) {
	smm := newStateMachineManager()

	smm.addEventHandler(eventTick, stateNew, func(m *stateMachine, i interface{}) (id stateID, err error) {
		return stateScheduled, nil
	})
	if len(smm.handlers[stateNew]) != 1 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.handlers[stateNew]), 1)
	}
	state, err := smm.handlers[stateNew][eventTick](nil, nil)
	if err != nil {
		t.Error(err)
	}
	if state != stateScheduled {
		t.Errorf("unexpected state, got: %v, want: %v", state, stateScheduled)
	}

	// Add second handler to the same event
	smm.addEventHandler(eventTick, stateSent, func(m *stateMachine, i interface{}) (id stateID, err error) {
		return stateDataParsed, nil
	})
	if len(smm.handlers[stateSent]) != 1 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.handlers[stateSent]), 1)
	}
	state, err = smm.handlers[stateSent][eventTick](nil, nil)
	if err != nil {
		t.Error(err)
	}
	if state != stateDataParsed {
		t.Errorf("unexpected state, got: %v, want: %v", state, stateScheduled)
	}

	// Add another handler to existing event/state pair. Should have no effect.
	smm.addEventHandler(eventTick, stateSent, func(m *stateMachine, i interface{}) (id stateID, err error) {
		return stateSkipped, nil
	})
	if len(smm.handlers[stateSent]) != 1 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.handlers[stateSent]), 1)
	}
	state, err = smm.handlers[stateSent][eventTick](nil, nil)
	if err != nil {
		t.Error(err)
	}
	// No effect, previous handler worked.
	if state != stateDataParsed {
		t.Errorf("unexpected state, got: %v, want: %v", state, stateScheduled)
	}
}

func TestStateMachine_trigger(t *testing.T) {
	type event struct {
		state       stateID
		event       eventID
		returnState stateID
		err         bool
	}
	type args struct {
		name        eventID
		returnState stateID
		epoch       uint64
		data        interface{}
	}
	tests := []struct {
		name   string
		events []event
		epochs []uint64
		args   args
		err    error
	}{
		{
			name:   "event not found",
			events: []event{},
			epochs: []uint64{12, 13},
			args:   args{name: eventTick, epoch: 12, data: nil, returnState: stateNew},
			err:    errors.New("no event handlers registered for event: tick, state: new"),
		},
		{
			name: "single action",
			events: []event{
				{stateNew, eventTick, stateScheduled, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventTick, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, has error",
			events: []event{
				{stateNew, eventTick, stateScheduled, false},
				{stateScheduled, eventTick, stateSent, true},
				{stateSent, eventTick, stateSkipped, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventTick, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, no error, can cascade",
			events: []event{
				{stateNew, eventTick, stateScheduled, false},
				{stateScheduled, eventTick, stateSent, false},
				{stateSent, eventTick, stateSkipped, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventTick, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, no error, no cascade",
			events: []event{
				{stateNew, eventTick, stateScheduled, false},
				{stateScheduled, eventTick, stateSent, false},
				{stateNew, eventTick, stateSkipped, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventTick, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
	}
	fn := func(e event) eventHandlerFn {
		return func(m *stateMachine, in interface{}) (stateID, error) {
			if e.err {
				return m.state, errors.New("invalid")
			}
			return e.returnState, nil
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smm := newStateMachineManager()
			expectHandlerError := false
			for _, event := range tt.events {
				smm.addEventHandler(event.event, event.state, fn(event))
				if event.err {
					expectHandlerError = true
				}
			}
			for _, epoch := range tt.epochs {
				smm.addStateMachine(epoch * 32)
			}
			state := smm.machines[tt.args.epoch*32]
			err := state.trigger(tt.args.name, tt.args.data)
			if tt.err != nil && (err == nil || tt.err.Error() != err.Error()) {
				t.Errorf("unexpected error = '%v', want '%v'", err, tt.err)
			}
			if tt.err == nil {
				if err != nil && !expectHandlerError {
					t.Error(err)
				}
				ind := tt.args.epoch * 32
				if smm.machines[ind].state != tt.args.returnState {
					t.Errorf("unexpected final state: %v, want: %v (%v)",
						smm.machines[ind].state, tt.args.returnState, smm.machines)
				}
			}
		})
	}
}

func TestStateMachineManager_QueueLoop(t *testing.T) {
	smm := newStateMachineManager()
	smm.addEventHandler(eventTick, stateNew, func(m *stateMachine, data interface{}) (stateID, error) {
		return stateScheduled, nil
	})
	smm.addEventHandler(eventTick, stateScheduled, func(m *stateMachine, data interface{}) (stateID, error) {
		if m.start < 256 {
			return stateDataParsed, nil
		}
		return stateSkipped, nil
	})
	smm.addEventHandler(eventTick, stateDataParsed, func(m *stateMachine, data interface{}) (stateID, error) {
		return stateSent, nil
	})
	smm.addEventHandler(eventTick, stateSkipped, func(m *stateMachine, data interface{}) (stateID, error) {
		dataParsed, ok := data.(int)
		if !ok {
			return m.state, errors.New("invalid data type")
		}
		if dataParsed > 41 {
			return stateNew, nil
		}

		return stateScheduled, nil
	})
	if len(smm.handlers) != 4 {
		t.Errorf("unexpected number of state events, want: %v, got: %v", 4, len(smm.handlers))
	}
	smm.addStateMachine(64)
	smm.addStateMachine(512)

	assertState := func(startBlock uint64, state stateID) {
		fsm, ok := smm.findStateMachine(startBlock)
		if !ok {
			t.Fatalf("state machine not found: %v", startBlock)
		}
		if fsm.state != state {
			t.Errorf("unexpected state machine state, want: %v, got: %v", state, fsm.state)
		}
	}

	triggerTickEvent := func() {
		for _, fsm := range smm.machines {
			data := 42
			if err := fsm.trigger(eventTick, data); err != nil {
				t.Error(err)
			}
		}
	}

	assertState(64, stateNew)
	assertState(512, stateNew)

	triggerTickEvent()
	assertState(64, stateScheduled)
	assertState(512, stateScheduled)

	triggerTickEvent()
	assertState(64, stateDataParsed)
	assertState(512, stateSkipped)

	triggerTickEvent()
	assertState(64, stateSent)
	assertState(512, stateNew)
}

func TestStateMachineManager_removeStateMachine(t *testing.T) {
	smm := newStateMachineManager()
	if _, ok := smm.findStateMachine(64); ok {
		t.Error("unexpected machine found")
	}
	smm.addStateMachine(64)
	if _, ok := smm.findStateMachine(64); !ok {
		t.Error("expected machine not found")
	}
	expectedError := fmt.Errorf("state for machine %v is not found", 65)
	if err := smm.removeStateMachine(65); err == nil || err.Error() != expectedError.Error() {
		t.Errorf("expected error (%v), got: %v", expectedError, err)
	}
	if err := smm.removeStateMachine(64); err != nil {
		t.Error(err)
	}
	if _, ok := smm.findStateMachine(64); ok {
		t.Error("unexpected machine found")
	}
}

func TestStateMachineManager_removeAllStateMachines(t *testing.T) {
	smm := newStateMachineManager()
	smm.addStateMachine(64)
	smm.addStateMachine(128)
	smm.addStateMachine(196)
	keys := []uint64{64, 128, 196}
	if !reflect.DeepEqual(keys, smm.keys) {
		t.Errorf("keys not sorted, want: %v, got: %v", keys, smm.keys)
	}
	if len(smm.machines) != 3 {
		t.Errorf("unexpected list size: %v", len(smm.machines))
	}

	if err := smm.removeAllStateMachines(); err != nil {
		t.Error(err)
	}

	keys = []uint64{}
	if !reflect.DeepEqual(keys, smm.keys) {
		t.Errorf("unexpected keys, want: %v, got: %v", keys, smm.keys)
	}
	if len(smm.machines) != 0 {
		t.Error("expected empty list")
	}
}

func TestStateMachineManager_findStateMachine(t *testing.T) {
	smm := newStateMachineManager()
	if _, ok := smm.findStateMachine(64); ok {
		t.Errorf("unexpected returned value: want: %v, got: %v", false, ok)
	}
	smm.addStateMachine(64)
	if fsm, ok := smm.findStateMachine(64); !ok || fsm == nil {
		t.Errorf("unexpected returned value: want: %v, got: %v", true, ok)
	}
	smm.addStateMachine(512)
	smm.addStateMachine(196)
	smm.addStateMachine(256)
	smm.addStateMachine(128)
	if fsm, ok := smm.findStateMachine(128); !ok || fsm.start != 128 {
		t.Errorf("unexpected start block: %v, want: %v", fsm.start, 122)
	}
	if fsm, ok := smm.findStateMachine(512); !ok || fsm.start != 512 {
		t.Errorf("unexpected start block: %v, want: %v", fsm.start, 512)
	}
	keys := []uint64{64, 128, 196, 256, 512}
	if !reflect.DeepEqual(keys, smm.keys) {
		t.Errorf("keys not sorted, want: %v, got: %v", keys, smm.keys)
	}
}

func TestStateMachineManager_highestStartBlock(t *testing.T) {
	smm := newStateMachineManager()
	if _, err := smm.highestStartBlock(); err == nil {
		t.Error("expected error")
	}
	smm.addStateMachine(64)
	smm.addStateMachine(128)
	smm.addStateMachine(196)
	start, err := smm.highestStartBlock()
	if err != nil {
		t.Error(err)
	}
	if start != 196 {
		t.Errorf("incorrect highest start block: %v, want: %v", start, 196)
	}
	if err := smm.removeStateMachine(196); err != nil {
		t.Error(err)
	}
	start, err = smm.highestStartBlock()
	if err != nil {
		t.Error(err)
	}
	if start != 128 {
		t.Errorf("incorrect highest start block: %v, want: %v", start, 128)
	}
}

func TestStateMachineManager_allMachinesInState(t *testing.T) {
	tests := []struct {
		name             string
		smmGen           func() *stateMachineManager
		expectedStates   []stateID
		unexpectedStates []stateID
	}{
		{
			name: "empty manager",
			smmGen: func() *stateMachineManager {
				return newStateMachineManager()
			},
			expectedStates:   []stateID{},
			unexpectedStates: []stateID{stateNew, stateScheduled, stateDataParsed, stateSkipped, stateSent},
		},
		{
			name: "single machine default state",
			smmGen: func() *stateMachineManager {
				smm := newStateMachineManager()
				smm.addStateMachine(64)
				return smm
			},
			expectedStates:   []stateID{stateNew},
			unexpectedStates: []stateID{stateScheduled, stateDataParsed, stateSkipped, stateSent},
		},
		{
			name: "single machine updated state",
			smmGen: func() *stateMachineManager {
				smm := newStateMachineManager()
				m1 := smm.addStateMachine(64)
				m1.setState(stateSkipped)
				return smm
			},
			expectedStates:   []stateID{stateSkipped},
			unexpectedStates: []stateID{stateNew, stateScheduled, stateDataParsed, stateSent},
		},
		{
			name: "multiple machines false",
			smmGen: func() *stateMachineManager {
				smm := newStateMachineManager()
				smm.addStateMachine(64)
				smm.addStateMachine(128)
				smm.addStateMachine(196)
				for _, fsm := range smm.machines {
					fsm.setState(stateSkipped)
				}
				smm.addStateMachine(256)
				return smm
			},
			expectedStates:   []stateID{},
			unexpectedStates: []stateID{stateNew, stateScheduled, stateDataParsed, stateSkipped, stateSent},
		},
		{
			name: "multiple machines true",
			smmGen: func() *stateMachineManager {
				smm := newStateMachineManager()
				smm.addStateMachine(64)
				smm.addStateMachine(128)
				smm.addStateMachine(196)
				for _, fsm := range smm.machines {
					fsm.setState(stateSkipped)
				}
				return smm
			},
			expectedStates:   []stateID{stateSkipped},
			unexpectedStates: []stateID{stateNew, stateScheduled, stateDataParsed, stateSent},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smm := tt.smmGen()
			for _, state := range tt.expectedStates {
				if !smm.allMachinesInState(state) {
					t.Errorf("expected all machines be in state: %v", state)
				}
			}
			for _, state := range tt.unexpectedStates {
				if smm.allMachinesInState(state) {
					t.Errorf("unexpected state: %v", state)
				}
			}
		})
	}
}

func TestStateMachine_isFirstLast(t *testing.T) {
	checkFirst := func(m *stateMachine, want bool) {
		if m.isFirst() != want {
			t.Errorf("isFirst() returned unexpected value, want: %v, got: %v", want, m.start)
		}
	}
	checkLast := func(m *stateMachine, want bool) {
		if m.isLast() != want {
			t.Errorf("isLast(%v) returned unexpected value, want: %v, got: %v", m.start, want, m.start)
		}
	}
	smm := newStateMachineManager()
	m1 := smm.addStateMachine(64)
	checkFirst(m1, true)
	checkLast(m1, true)

	m2 := smm.addStateMachine(128)
	checkFirst(m1, true)
	checkLast(m1, false)
	checkFirst(m2, false)
	checkLast(m2, true)

	m3 := smm.addStateMachine(512)
	checkFirst(m1, true)
	checkLast(m1, false)
	checkFirst(m2, false)
	checkLast(m2, false)
	checkFirst(m3, false)
	checkLast(m3, true)

	// Add machine with lower start block - shouldn't be marked as last.
	m4 := smm.addStateMachine(196)
	checkFirst(m1, true)
	checkLast(m1, false)
	checkFirst(m2, false)
	checkLast(m2, false)
	checkFirst(m3, false)
	checkLast(m3, true)
	checkFirst(m4, false)
	checkLast(m4, false)

	// Add machine with lowest start block - should be marked as first.
	m5 := smm.addStateMachine(32)
	checkFirst(m1, false)
	checkLast(m1, false)
	checkFirst(m2, false)
	checkLast(m2, false)
	checkFirst(m3, false)
	checkLast(m3, true)
	checkFirst(m4, false)
	checkLast(m4, false)
	checkFirst(m5, true)
	checkLast(m5, false)

	keys := []uint64{32, 64, 128, 196, 512}
	if !reflect.DeepEqual(keys, smm.keys) {
		t.Errorf("keys not sorted, want: %v, got: %v", keys, smm.keys)
	}
}
