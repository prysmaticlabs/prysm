package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

var _ = log.Handler(&TestLogHandler{})

// TestLogHandler provides methods for testing ethereum logs.
type TestLogHandler struct {
	records []*log.Record
	t       *testing.T
}

// Log adds records to the record slice.
func (t *TestLogHandler) Log(r *log.Record) error {
	t.records = append(t.records, r)
	return nil
}

// Pop the record at index 0.
func (t *TestLogHandler) Pop() *log.Record {
	var r *log.Record
	r, t.records = t.records[0], t.records[1:]
	return r
}

// Length of the records slice.
func (t *TestLogHandler) Len() int {
	return len(t.records)
}

// VerifyLogMsg verfies that the log at index 0 matches the string exactly.
// This method removes the verified message from the slice.
func (h *TestLogHandler) VerifyLogMsg(str string) {
	if h.Len() == 0 {
		h.t.Error("Expected a log, but there were none!")
	}
	if l := h.Pop(); l.Msg != str {
		h.t.Errorf("Unexpected log: %v. Wanted: %s", l, str)
	}
}
