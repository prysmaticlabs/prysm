package initialsync

import (
	"testing"
)

func BenchmarkStateMachine_trigger(b *testing.B) {
	sm := newStateMachine()

	handlerFn := func(state *epochState, in interface{}) (id stateID, err error) {
		response, ok := in.(*fetchRequestParams)
		if !ok {
			return 0, errInputNotFetchRequestParams
		}
		_ = response.count
		return stateScheduled, nil
	}

	sm.addHandler(stateNew, eventSchedule, handlerFn)
	sm.addHandler(stateScheduled, eventDataReceived, handlerFn)
	sm.addHandler(stateDataParsed, eventReadyToSend, handlerFn)
	sm.addHandler(stateSkipped, eventExtendWindow, handlerFn)
	sm.addHandler(stateSent, eventCheckStale, handlerFn)

	for i := uint64(0); i < lookaheadEpochs; i++ {
		sm.addEpochState(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	event, ok := sm.events[eventSchedule]
	if !ok {
		b.Errorf("event not found: %v", eventSchedule)
	}

	for i := 0; i < b.N; i++ {
		data := &fetchRequestParams{
			start: 23,
			count: 32,
		}
		err := sm.epochs[1].trigger(event, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
