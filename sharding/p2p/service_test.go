package p2p

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
)

// Ensure that server implements service.
var _ = sharding.Service(&Server{})

// TODO: Move test log handler to another test package.
type testLogHandler struct {
	records []*log.Record
	t       *testing.T
}

func (t *testLogHandler) Log(r *log.Record) error {
	t.records = append(t.records, r)
	return nil
}

func (t *testLogHandler) Pop() *log.Record {
	var r *log.Record
	r, t.records = t.records[0], t.records[1:]
	return r
}

func (t *testLogHandler) Len() int {
	return len(t.records)
}

func (h *testLogHandler) verifyLogMsg(str string) {
	if h.Len() == 0 {
		h.t.Error("Expected a log, but there were none!")
	}
	if l := h.Pop(); l.Msg != str {
		h.t.Errorf("Unexpected log: %v. Wanted: %s", l, str)
	}
}

func TestLifecycle(t *testing.T) {
	h := &testLogHandler{t: t}
	logger.SetHandler(h)

	s, err := NewServer()
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	s.Start()
	h.verifyLogMsg("Starting shardp2p server")

	s.Stop()
	h.verifyLogMsg("Stopping shardp2p server")

	// The context should have been cancelled.
	if s.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}
