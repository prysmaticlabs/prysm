package initialsync

import (
	"errors"
	"testing"
)

func TestStateMachineManager_String(t *testing.T) {
	tests := []struct {
		name     string
		machines []*stateMachine
		want     string
	}{
		{
			"empty epoch state list",
			[]*stateMachine{},
			"[]",
		},
		{
			"newly created state machine",
			[]*stateMachine{
				{start: 8, state: stateNew},
				{start: 9, state: stateScheduled},
				{start: 10, state: stateDataParsed},
				{start: 11, state: stateSkipped},
				{start: 12, state: stateSkippedExt},
				{start: 13, state: stateComplete},
				{start: 14, state: stateSent},
			},
			"[8:new 9:scheduled 10:dataParsed 11:skipped 12:skippedExt 13:complete 14:sent]",
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

func TestStateMachineManager_addEventHandler(t *testing.T) {
	smm := newStateMachineManager()

	smm.addEventHandler(eventSchedule, stateNew, func(m *stateMachine, i interface{}) (id stateID, err error) {
		return stateScheduled, nil
	})
	if len(smm.events[eventSchedule].actions) != 1 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.events[eventSchedule].actions), 1)
	}
	state, err := smm.events[eventSchedule].actions[stateNew][0](nil, nil)
	if err != nil {
		t.Error(err)
	}
	if state != stateScheduled {
		t.Errorf("unexpected state, got: %v, want: %v", state, stateScheduled)
	}

	// Add second handler to the same event
	smm.addEventHandler(eventSchedule, stateSent, func(m *stateMachine, i interface{}) (id stateID, err error) {
		return stateDataParsed, nil
	})
	if len(smm.events[eventSchedule].actions) != 2 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.events[eventSchedule].actions), 2)
	}
	state, err = smm.events[eventSchedule].actions[stateSent][0](nil, nil)
	if err != nil {
		t.Error(err)
	}
	if state != stateDataParsed {
		t.Errorf("unexpected state, got: %v, want: %v", state, stateScheduled)
	}

	// Add another handler to existing event/state pair.
	smm.addEventHandler(eventSchedule, stateSent, func(m *stateMachine, i interface{}) (id stateID, err error) {
		return stateSkippedExt, nil
	})
	if len(smm.events[eventSchedule].actions) != 2 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.events[eventSchedule].actions), 2)
	}
	if len(smm.events[eventSchedule].actions[stateSent]) != 2 {
		t.Errorf("unexpected size, got: %v, want: %v", len(smm.events[eventSchedule].actions[stateSent]), 2)
	}
	state, err = smm.events[eventSchedule].actions[stateSent][0](nil, nil)
	if err != nil {
		t.Error(err)
	}
	// The action that changes state to stateSkippedExt will not be triggered, as previous action
	// changed state - and machine's state is not applicable anymore.
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
			args:   args{name: eventSchedule, epoch: 12, data: nil, returnState: stateNew},
			err:    errors.New("state machine or event is nil"),
		},
		{
			name: "single action",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventSchedule, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, has error",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
				{stateScheduled, eventSchedule, stateSent, true},
				{stateSent, eventSchedule, stateComplete, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventSchedule, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, no error, can cascade",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
				{stateScheduled, eventSchedule, stateSent, false},
				{stateSent, eventSchedule, stateComplete, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventSchedule, epoch: 12, data: nil, returnState: stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, no error, no cascade",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
				{stateScheduled, eventSchedule, stateSent, false},
				{stateNew, eventSchedule, stateComplete, false},
			},
			epochs: []uint64{12, 13},
			args:   args{name: eventSchedule, epoch: 12, data: nil, returnState: stateScheduled},
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
				smm.addStateMachine(epoch*32, 64)
			}
			ind, ok := smm.findStateMachineByStartBlock(tt.args.epoch * 32)
			if !ok {
				t.Fatal("machine state not found")
			}
			state := smm.machines[ind]
			err := state.trigger(smm.events[tt.args.name], tt.args.data)
			if tt.err != nil && (err == nil || tt.err.Error() != err.Error()) {
				t.Errorf("unexpected error = '%v', want '%v'", err, tt.err)
			}
			if tt.err == nil {
				if err != nil && !expectHandlerError {
					t.Error(err)
				}
				ind, ok := smm.findStateMachineByStartBlock(tt.args.epoch * 32)
				if !ok {
					t.Errorf("expected machine not found, start block: %v", tt.args.epoch*32)
					return
				}
				if smm.machines[ind].state != tt.args.returnState {
					t.Errorf("unexpected final state: %v, want: %v (%v)",
						smm.machines[ind].state, tt.args.returnState, smm.machines)
				}
			}
		})
	}
}

//
//func TestStateMachine_findEpochState(t *testing.T) {
//	smm := newStateMachineManager()
//	if ind, ok := smm.findEpochState(12); ok || ind != 0 {
//		t.Errorf("unexpected index: %v, want: %v", ind, 0)
//	}
//	smm.addEpochState(12)
//	if ind, ok := smm.findEpochState(12); !ok || ind != 0 {
//		t.Errorf("unexpected index: %v, want: %v", ind, 0)
//	}
//	smm.addEpochState(13)
//	smm.addEpochState(14)
//	smm.addEpochState(15)
//	if ind, ok := smm.findEpochState(14); !ok || ind != 2 {
//		t.Errorf("unexpected index: %v, want: %v", ind, 2)
//	}
//	if ind, ok := smm.findEpochState(16); ok || ind != len(smm.epochs) {
//		t.Errorf("unexpected index: %v, want: %v", ind, len(smm.epochs))
//	}
//}
//
//func TestStateMachine_isLowestEpochState(t *testing.T) {
//	smm := newStateMachineManager()
//	smm.addEpochState(12)
//	smm.addEpochState(13)
//	smm.addEpochState(14)
//	if res := smm.isLowestEpochState(15); res {
//		t.Errorf("unexpected lowest state: %v", 15)
//	}
//	if res := smm.isLowestEpochState(13); res {
//		t.Errorf("unexpected lowest state: %v", 15)
//	}
//	if res := smm.isLowestEpochState(12); !res {
//		t.Errorf("expected lowest state not found: %v", 12)
//	}
//	if err := smm.removeEpochState(12); err != nil {
//		t.Error(err)
//	}
//	if res := smm.isLowestEpochState(12); res {
//		t.Errorf("unexpected lowest state: %v", 12)
//	}
//	if res := smm.isLowestEpochState(13); !res {
//		t.Errorf("expected lowest state not found: %v", 13)
//	}
//}
//
//func TestStateMachine_highestEpoch(t *testing.T) {
//	smm := newStateMachineManager()
//	if _, err := smm.highestEpoch(); err == nil {
//		t.Error("expected error")
//	}
//	smm.addEpochState(12)
//	smm.addEpochState(13)
//	smm.addEpochState(14)
//	epoch, err := smm.highestEpoch()
//	if err != nil {
//		t.Error(err)
//	}
//	if epoch != 14 {
//		t.Errorf("incorrect highest epoch: %v, want: %v", epoch, 14)
//	}
//	if err := smm.removeEpochState(14); err != nil {
//		t.Error(err)
//	}
//	epoch, err = smm.highestEpoch()
//	if err != nil {
//		t.Error(err)
//	}
//	if epoch != 13 {
//		t.Errorf("incorrect highest epoch: %v, want: %v", epoch, 13)
//	}
//}
//
//func TestStateMachine_lowestEpoch(t *testing.T) {
//	smm := newStateMachineManager()
//	if _, err := smm.highestEpoch(); err == nil {
//		t.Error("expected error")
//	}
//	smm.addEpochState(12)
//	smm.addEpochState(13)
//	smm.addEpochState(14)
//	epoch, err := smm.lowestEpoch()
//	if err != nil {
//		t.Error(err)
//	}
//	if epoch != 12 {
//		t.Errorf("incorrect highest epoch: %v, want: %v", epoch, 12)
//	}
//	if err := smm.removeEpochState(12); err != nil {
//		t.Error(err)
//	}
//	epoch, err = smm.lowestEpoch()
//	if err != nil {
//		t.Error(err)
//	}
//	if epoch != 13 {
//		t.Errorf("incorrect highest epoch: %v, want: %v", epoch, 13)
//	}
//}
