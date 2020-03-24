package initialsync

import (
	"errors"
	"fmt"
	"testing"
)

func TestStateMachine_Stringify(t *testing.T) {
	tests := []struct {
		name   string
		epochs []*epochState
		want   string
	}{
		{
			"empty epoch state list",
			make([]*epochState, 0, lookaheadEpochs),
			"[]",
		},
		{
			"newly created state machine",
			[]*epochState{
				{epoch: 7, state: stateNew,},
				{epoch: 8, state: stateScheduled,},
				{epoch: 9, state: stateDataReceived,},
				{epoch: 10, state: stateDataParsed,},
				{epoch: 11, state: stateSkipped,},
				{epoch: 12, state: stateSkippedExt,},
				{epoch: 13, state: stateComplete,},
				{epoch: 14, state: stateSent,},
			},
			"[7:new 8:scheduled 9:dataReceived 10:dataParsed 11:skipped 12:skippedExt 13:complete 14:sent]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &stateMachine{
				epochs: tt.epochs,
			}
			if got := sm.String(); got != tt.want {
				t.Errorf("unexpected output,  got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestStateMachine_addHandler(t *testing.T) {
	sm := newStateMachine()

	sm.addHandler(stateNew, eventSchedule, func(state *epochState, i interface{}) (id stateID, err error) {
		return stateScheduled, nil
	})
	if len(sm.events[eventSchedule].actions) != 1 {
		t.Errorf("unexpected size, got: %v, want: %v", len(sm.events[eventSchedule].actions), 1)
	}
	state, err := sm.events[eventSchedule].actions[0].handlerFn(nil, nil)
	if err != nil {
		t.Error(err)
	}
	if state != stateScheduled {
		t.Errorf("unexpected state, got: %v, want: %v", state, stateScheduled)
	}

	// Add second handler to the same event
	sm.addHandler(stateSent, eventSchedule, func(state *epochState, i interface{}) (id stateID, err error) {
		return stateDataReceived, nil
	})
	if len(sm.events[eventSchedule].actions) != 2 {
		t.Errorf("unexpected size, got: %v, want: %v", len(sm.events[eventSchedule].actions), 2)
	}
	state, err = sm.events[eventSchedule].actions[1].handlerFn(nil, nil)
	if err != nil {
		t.Error(err)
	}
	if state != stateDataReceived {
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
		epoch       uint64
		data        interface{}
		returnState stateID
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
			epochs: []uint64{},
			args:   args{eventSchedule, 12, nil, stateNew},
			err:    fmt.Errorf("event not found: %v", eventSchedule),
		},
		{
			name: "epoch not found",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
			},
			epochs: []uint64{},
			args:   args{eventSchedule, 12, nil, stateScheduled},
			err:    fmt.Errorf("state for %v epoch not found", 12),
		},
		{
			name: "single action",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
			},
			epochs: []uint64{12, 13},
			args:   args{eventSchedule, 12, nil, stateScheduled},
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
			args:   args{eventSchedule, 12, nil, stateScheduled},
			err:    nil,
		},
		{
			name: "multiple actions, no error, cascade",
			events: []event{
				{stateNew, eventSchedule, stateScheduled, false},
				{stateScheduled, eventSchedule, stateSent, false},
				{stateSent, eventSchedule, stateComplete, false},
			},
			epochs: []uint64{12, 13},
			args:   args{eventSchedule, 12, nil, stateComplete},
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
			args:   args{eventSchedule, 12, nil, stateSent},
			err:    nil,
		},
	}
	fn := func(e event) eventHandlerFn {
		return func(es *epochState, in interface{}) (stateID, error) {
			if e.err {
				return es.state, errors.New("invalid")
			}
			return e.returnState, nil
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := newStateMachine()
			expectHandlerError := false
			for _, event := range tt.events {
				sm.addHandler(event.state, event.event, fn(event))
				if event.err {
					expectHandlerError = true
				}
			}
			for _, epoch := range tt.epochs {
				sm.addEpochState(epoch)
			}
			err := sm.trigger(tt.args.name, tt.args.epoch, tt.args.data)
			if tt.err != nil && (err == nil || tt.err.Error() != err.Error()) {
				t.Errorf("unexpected error = '%v', want '%v'", err, tt.err)
			}
			if tt.err == nil {
				if err != nil && !expectHandlerError {
					t.Error(err)
				}
				ind := sm.findEpochState(tt.args.epoch)
				if ind >= len(sm.epochs) {
					t.Errorf("expected epoch not found: %v", tt.args.epoch)
					return
				}
				if sm.epochs[ind].state != tt.args.returnState {
					t.Errorf("unexpected final state: %v, want: %v (%v)", sm.epochs[ind].state, tt.args.returnState, sm.epochs)
				}
			}
		})
	}
}

func TestStateMachine_findEpochState(t *testing.T) {
	sm := newStateMachine()
	if ind := sm.findEpochState(12); ind != 0 {
		t.Errorf("unexpected index: %v, want: %v", ind, 0)
	}
	sm.addEpochState(12)
	if ind := sm.findEpochState(12); ind != 0 {
		t.Errorf("unexpected index: %v, want: %v", ind, 0)
	}
	sm.addEpochState(13)
	sm.addEpochState(14)
	sm.addEpochState(15)
	if ind := sm.findEpochState(14); ind != 2 {
		t.Errorf("unexpected index: %v, want: %v", ind, 2)
	}
	if ind := sm.findEpochState(16); ind != len(sm.epochs) {
		t.Errorf("unexpected index: %v, want: %v", ind, len(sm.epochs))
	}
}

func TestStateMachine_isLowestEpochState(t *testing.T) {
	sm := newStateMachine()
	sm.addEpochState(12)
	sm.addEpochState(13)
	sm.addEpochState(14)
	if res := sm.isLowestEpochState(15); res {
		t.Errorf("unexpected lowest state: %v", 15)
	}
	if res := sm.isLowestEpochState(13); res {
		t.Errorf("unexpected lowest state: %v", 15)
	}
	if res := sm.isLowestEpochState(12); !res {
		t.Errorf("expected lowest state not found: %v", 12)
	}
	if err := sm.removeEpochState(12); err != nil {
		t.Error(err)
	}
	if res := sm.isLowestEpochState(12); res {
		t.Errorf("unexpected lowest state: %v", 12)
	}
	if res := sm.isLowestEpochState(13); !res {
		t.Errorf("expected lowest state not found: %v", 13)
	}
}

func TestStateMachine_highestEpochSlot(t *testing.T) {
	sm := newStateMachine()
	if _, err := sm.highestEpochSlot(); err == nil {
		t.Error("expected error")
	}
	sm.addEpochState(12)
	sm.addEpochState(13)
	sm.addEpochState(14)
	slot, err := sm.highestEpochSlot()
	if err != nil {
		t.Error(err)
	}
	if slot != 14 {
		t.Errorf("incorrect highest slot: %v, want: %v", slot, 14)
	}
	if err := sm.removeEpochState(14); err != nil {
		t.Error(err)
	}
	slot, err = sm.highestEpochSlot()
	if err != nil {
		t.Error(err)
	}
	if slot != 13 {
		t.Errorf("incorrect highest slot: %v, want: %v", slot, 13)
	}
}
