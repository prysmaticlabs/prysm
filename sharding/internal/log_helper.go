package utils

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

var _ = log.Handler(&LogHandler{})

// LogHandler provides methods for testing ethereum logs.
type LogHandler struct {
	records     []*log.Record
	t           *testing.T
	mutex       sync.Mutex
	logChanOpen bool
	logWritten  chan bool
}

func NewLogHandler(t *testing.T) *LogHandler {
	return &LogHandler{t: t}
}

// Log adds records to the record slice.
func (t *LogHandler) Log(r *log.Record) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.records = append(t.records, r)
	if t.logChanOpen {
		t.logWritten <- true
	}
	return nil
}

func (t *LogHandler) WaitForLog() {
	t.mutex.Lock()
	t.logWritten = make(chan bool)
	t.logChanOpen = true
	defer close(t.logWritten)
	t.mutex.Unlock()
	<-t.logWritten
	t.logChanOpen = false
}

// Pop the record at index 0.
func (t *LogHandler) Pop() *log.Record {
	var r *log.Record
	r, t.records = t.records[0], t.records[1:]
	return r
}

// Length of the records slice.
func (t *LogHandler) Len() int {
	return len(t.records)
}

// VerifyLogMsg verfies that the log at index 0 matches the string exactly.
// This method removes the verified message from the slice.
func (h *LogHandler) VerifyLogMsg(str string) {
	if h.Len() == 0 {
		h.t.Error("Expected a log, but there were none!")
	}
	if l := h.Pop(); l.Msg != str {
		h.t.Errorf("Unexpected log: %v. Wanted: %s", l.Msg, str)
	}
}
