package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

var _ = log.Handler(&LogHandler{})

// LogHandler provides methods for testing ethereum logs.
type LogHandler struct {
	records []*log.Record
	t       *testing.T
}

func NewLogHandler(t *testing.T) *LogHandler {
	return &LogHandler{t: t}
}

// Log adds records to the record slice.
func (t *LogHandler) Log(r *log.Record) error {
	t.records = append(t.records, r)
	return nil
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
